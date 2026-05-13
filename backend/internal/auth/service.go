package auth

import (
    "encoding/gob"
    "net/http"
    "strings"
    "time"

    "github.com/alexedwards/scs/v2"
    "golang.org/x/crypto/bcrypt"

    "slatessh/backend/internal/users"
)

type Service struct {
    users    *users.Repository
    sessions *scs.SessionManager
}

type SessionUser struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
}

func init() {
    gob.Register(SessionUser{})
}

func NewService(users *users.Repository, sessions *scs.SessionManager) *Service {
    return &Service{users: users, sessions: sessions}
}

func (s *Service) NeedsSetup() (bool, error) {
    count, err := s.users.Count()
    return count == 0, err
}

func (s *Service) SetupAdmin(username, password, confirmPassword string) error {
    if strings.TrimSpace(username) == "" || password == "" || confirmPassword == "" {
        return errBadRequest("请填写完整的用户名和密码。")
    }
    if password != confirmPassword {
        return errBadRequest("两次输入的密码不一致。")
    }
    count, err := s.users.Count()
    if err != nil {
        return err
    }
    if count > 0 {
        return errConflict("系统已完成初始化。")
    }
    hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    _, err = s.users.Create(strings.TrimSpace(username), string(hashed))
    return err
}

func (s *Service) Login(_ http.ResponseWriter, r *http.Request, username, password string, rememberMe bool) (*SessionUser, error) {
    if strings.TrimSpace(username) == "" || password == "" {
        return nil, errBadRequest("请输入用户名和密码。")
    }
    user, err := s.users.FindByUsername(strings.TrimSpace(username))
    if err != nil {
        return nil, err
    }
    if user == nil || bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)) != nil {
        return nil, errUnauthorized("用户名或密码错误。")
    }

    sessionUser := SessionUser{ID: user.ID, Username: user.Username}
    s.sessions.Put(r.Context(), "user", sessionUser)
    if rememberMe {
        s.sessions.SetDeadline(r.Context(), time.Now().Add(365*24*time.Hour))
    }
    return &sessionUser, nil
}

func (s *Service) Logout(r *http.Request) error {
    return s.sessions.Destroy(r.Context())
}

func (s *Service) CurrentUser(r *http.Request) *SessionUser {
    if !s.sessions.Exists(r.Context(), "user") {
        return nil
    }
    user, ok := s.sessions.Get(r.Context(), "user").(SessionUser)
    if !ok {
        return nil
    }
    return &user
}

func (s *Service) ChangePassword(r *http.Request, currentPassword, newPassword string) error {
    currentUser := s.CurrentUser(r)
    if currentUser == nil {
        return errUnauthorized("未授权。")
    }
    user, err := s.users.FindByID(currentUser.ID)
    if err != nil {
        return err
    }
    if user == nil {
        return errUnauthorized("用户不存在。")
    }
    if bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(currentPassword)) != nil {
        return errUnauthorized("当前密码错误。")
    }
    hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    return s.users.UpdatePassword(user.ID, string(hashed))
}

type appError struct {
    Message string
    Status  int
}

func (e appError) Error() string { return e.Message }

func errBadRequest(message string) error { return appError{Message: message, Status: http.StatusBadRequest} }
func errUnauthorized(message string) error { return appError{Message: message, Status: http.StatusUnauthorized} }
func errConflict(message string) error { return appError{Message: message, Status: http.StatusConflict} }


