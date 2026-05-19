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

// init 用于执行 init 相关后端逻辑。
// 输入参数：无。
// 输出参数：无。
func init() {
	gob.Register(SessionUser{})
}

// NewService 用于创建业务服务实例。
// 输入参数：users 表示users 参数；sessions 表示sessions 参数。
// 输出参数：返回 *Service。
func NewService(users *users.Repository, sessions *scs.SessionManager) *Service {
	return &Service{users: users, sessions: sessions}
}

// NeedsSetup 用于返回系统是否需要初始化管理员账号。
// 输入参数：无。
// 输出参数：返回 bool, error；error 表示执行失败原因。
func (s *Service) NeedsSetup() (bool, error) {
	count, err := s.users.Count()
	return count == 0, err
}

// SetupAdmin 用于处理初始化管理员账号请求。
// 输入参数：username 表示用户名；password 表示密码；confirmPassword 表示confirmPassword 参数。
// 输出参数：返回 error；error 表示执行失败原因。
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

// Login 用于处理用户登录请求并写入会话。
// 输入参数：r 表示HTTP 请求对象；username 表示用户名；password 表示密码；rememberMe 表示rememberMe 参数。
// 输出参数：返回 *SessionUser, error；error 表示执行失败原因。
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

// Logout 用于处理用户退出登录请求并清理会话。
// 输入参数：r 表示HTTP 请求对象。
// 输出参数：返回 error；error 表示执行失败原因。
func (s *Service) Logout(r *http.Request) error {
	return s.sessions.Destroy(r.Context())
}

// CurrentUser 用于执行 CurrentUser 相关后端逻辑。
// 输入参数：r 表示HTTP 请求对象。
// 输出参数：返回 *SessionUser。
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

// ChangePassword 用于处理当前用户修改密码请求。
// 输入参数：r 表示HTTP 请求对象；currentPassword 表示currentPassword 参数；newPassword 表示新密码。
// 输出参数：返回 error；error 表示执行失败原因。
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

// Error 用于执行 Error 相关后端逻辑。
// 输入参数：无。
// 输出参数：返回 string。
func (e appError) Error() string { return e.Message }

// errBadRequest 用于创建请求参数错误。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error { return appError；error 表示执行失败原因。
func errBadRequest(message string) error {
	return appError{Message: message, Status: http.StatusBadRequest}
}

// errUnauthorized 用于创建未授权错误。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error { return appError；error 表示执行失败原因。
func errUnauthorized(message string) error {
	return appError{Message: message, Status: http.StatusUnauthorized}
}

// errConflict 用于创建资源冲突错误。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error { return appError；error 表示执行失败原因。
func errConflict(message string) error {
	return appError{Message: message, Status: http.StatusConflict}
}
