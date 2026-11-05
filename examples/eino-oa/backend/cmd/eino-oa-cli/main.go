// eino-oa-cli 是面向 AI Agent / 命令行的授权登录工具。
//
// 它使用 OAuth2 Authorization Code + PKCE（本机回环回调）：
// 打开浏览器 → 用户在系统里确认授权 → CLI 在 127.0.0.1 收到授权码 → 换取短期令牌。
//
// 用法：
//
//	eino-oa-cli login --server http://localhost:8080
//	eino-oa-cli token          # 输出 access token，供脚本 / Agent 使用
//	eino-oa-cli logout
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const clientID = "eino-oa-cli"

type credentials struct {
	Server       string    `json:"server"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "login":
		err = runLogin(os.Args[2:])
	case "token":
		err = runToken()
	case "logout":
		err = runLogout()
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `eino-oa-cli - 学生管理系统授权工具

用法:
  eino-oa-cli login [--server URL] [--scope read,write] [--no-browser]
  eino-oa-cli token
  eino-oa-cli logout

说明:
  login   打开浏览器完成授权，凭证保存在 ~/.eino-oa/credentials.json
  token   输出访问令牌；过期时自动用 refresh token 续期
  logout  删除本地凭证（不会吊销服务端令牌）

环境变量:
  EINO_OA_SERVER       默认服务地址，默认 http://localhost:8080
  EINO_OA_CREDENTIALS  凭证文件路径
`)
}

func runLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	defaultServer := envOr("EINO_OA_SERVER", "http://localhost:8080")
	server := fs.String("server", defaultServer, "服务地址，例如 http://localhost:8080")
	scope := fs.String("scope", "read,write", "申请的权限范围：read 或 read,write")
	noBrowser := fs.Bool("no-browser", false, "只打印授权链接，不自动打开浏览器")
	_ = fs.Parse(args)

	base := normalizeServer(*server)

	verifier, challenge, err := newPKCE()
	if err != nil {
		return err
	}
	state, err := randomToken(16)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("无法监听本机回调端口: %w", err)
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", listener.Addr().(*net.TCPAddr).Port)

	resultCh := make(chan struct {
		code string
		err  error
	}, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "state 校验失败", http.StatusBadRequest)
			resultCh <- struct {
				code string
				err  error
			}{"", fmt.Errorf("state 校验失败")}
			return
		}
		if e := q.Get("error"); e != "" {
			http.Error(w, "授权失败: "+e, http.StatusBadRequest)
			resultCh <- struct {
				code string
				err  error
			}{"", fmt.Errorf("授权失败: %s %s", e, q.Get("error_description"))}
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "缺少授权码", http.StatusBadRequest)
			resultCh <- struct {
				code string
				err  error
			}{"", fmt.Errorf("回调缺少 code")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>授权成功</title>
<style>body{font-family:system-ui,-apple-system,"PingFang SC",sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f6f7f9}.c{background:#fff;padding:36px 44px;border-radius:14px;box-shadow:0 12px 40px rgba(0,0,0,.08);text-align:center}h1{font-size:20px;margin:0 0 10px}p{color:#666;margin:0}</style>
</head><body><div class="c"><h1>授权成功</h1><p>可以关闭此页面，回到命令行继续操作。</p></div></body></html>`)
		resultCh <- struct {
			code string
			err  error
		}{code, nil}
	})

	httpServer := &http.Server{Handler: mux}
	go func() { _ = httpServer.Serve(listener) }()
	defer httpServer.Close()

	authURL := base + "/oauth/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {*scope},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode()

	fmt.Fprintln(os.Stderr, "请在浏览器中完成授权：")
	fmt.Fprintln(os.Stderr, authURL)
	if !*noBrowser {
		if err := openBrowser(authURL); err != nil {
			fmt.Fprintf(os.Stderr, "（自动打开浏览器失败，请手动复制上面的链接）\n")
		}
	}
	fmt.Fprintln(os.Stderr, "等待授权中...")

	var code string
	select {
	case res := <-resultCh:
		if res.err != nil {
			return res.err
		}
		code = res.code
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("授权超时，请重试")
	}

	pair, err := exchangeCode(base+"/oauth/token", code, redirectURI, verifier)
	if err != nil {
		return err
	}

	creds := &credentials{
		Server:       base,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(pair.ExpiresIn) * time.Second),
		Scope:        pair.Scope,
	}
	path, err := saveCredentials(creds)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "授权成功，凭证已保存到 %s\n", path)
	fmt.Fprintln(os.Stderr, `提示：在 Agent 环境中可以使用 EINO_OA_API_TOKEN="$(eino-oa-cli token)"`)
	fmt.Println(pair.AccessToken)
	return nil
}

func runToken() error {
	creds, path, err := loadCredentials()
	if err != nil {
		return err
	}
	if time.Until(creds.ExpiresAt) < time.Minute {
		if creds.RefreshToken == "" {
			return fmt.Errorf("访问令牌已过期，请重新运行 eino-oa-cli login")
		}
		pair, err := refreshToken(creds.Server+"/oauth/token", creds.RefreshToken)
		if err != nil {
			return fmt.Errorf("刷新令牌失败，请重新 login: %w", err)
		}
		creds.AccessToken = pair.AccessToken
		creds.RefreshToken = pair.RefreshToken
		creds.ExpiresAt = time.Now().Add(time.Duration(pair.ExpiresIn) * time.Second)
		creds.Scope = pair.Scope
		if err := writeCredentials(path, creds); err != nil {
			return err
		}
	}
	fmt.Println(creds.AccessToken)
	return nil
}

func runLogout() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Fprintln(os.Stderr, "已删除本地凭证（服务端令牌仍可在网页「访问令牌」中吊销）")
	return nil
}

// ---------- OAuth / HTTP ----------

func exchangeCode(endpoint, code, redirectURI, verifier string) (*tokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}
	return postToken(endpoint, form)
}

func refreshToken(endpoint, token string) (*tokenResponse, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {token},
	}
	return postToken(endpoint, form)
}

func postToken(endpoint string, form url.Values) (*tokenResponse, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.PostForm(endpoint, form)
	if err != nil {
		return nil, fmt.Errorf("请求 token 端点失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &e)
		if e.Error != "" {
			return nil, fmt.Errorf("%s: %s", e.Error, e.Description)
		}
		return nil, fmt.Errorf("token 端点返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var pair tokenResponse
	if err := json.Unmarshal(body, &pair); err != nil {
		return nil, fmt.Errorf("解析 token 响应失败: %w", err)
	}
	if pair.AccessToken == "" {
		return nil, fmt.Errorf("token 响应缺少 access_token")
	}
	return &pair, nil
}

// ---------- PKCE ----------

func newPKCE() (verifier, challenge string, err error) {
	verifier, err = randomToken(48)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ---------- 凭证文件 ----------

func credentialsPath() (string, error) {
	if p := os.Getenv("EINO_OA_CREDENTIALS"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".eino-oa", "credentials.json"), nil
}

func loadCredentials() (*credentials, string, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, path, fmt.Errorf("未找到本地凭证，请先运行 eino-oa-cli login")
		}
		return nil, path, err
	}
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, path, fmt.Errorf("读取凭证失败: %w", err)
	}
	if c.AccessToken == "" {
		return nil, path, fmt.Errorf("凭证文件缺少 access_token，请重新 login")
	}
	return &c, path, nil
}

func saveCredentials(c *credentials) (string, error) {
	path, err := credentialsPath()
	if err != nil {
		return "", err
	}
	if err := writeCredentials(path, c); err != nil {
		return "", err
	}
	return path, nil
}

func writeCredentials(path string, c *credentials) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// 先写临时文件再 rename，避免进程中断留下半截凭证；权限固定为 0600。
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---------- 杂项 ----------

func normalizeServer(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "http://localhost:8080"
	}
	return strings.TrimRight(s, "/")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func openBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Start()
}
