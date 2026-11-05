package handler

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wuhaitao/eino-oa/backend/internal/auth"
)

// 目前是公开客户端 + PKCE，不保存 client_secret。
// 仅允许本机回环地址回调，降低授权码被截获的风险。
var oauthClients = map[string]bool{
	"eino-oa-cli":   true,
	"eino-oa-mcp":   true,
	"eino-oa-agent": true,
}

type OAuthHandler struct {
	svc *auth.Service
}

func NewOAuthHandler(svc *auth.Service) *OAuthHandler {
	return &OAuthHandler{svc: svc}
}

func (h *OAuthHandler) Register(r *gin.Engine) {
	r.GET("/oauth/authorize", h.svc.OptionalSession(), h.Authorize)
	r.POST("/oauth/authorize", h.svc.OptionalSession(), h.Approve)
	r.POST("/oauth/token", h.Token)
	r.GET("/.well-known/oauth-authorization-server", h.Metadata)
}

type authorizeParams struct {
	ClientID     string
	RedirectURI  string
	ResponseType string
	Challenge    string
	State        string
	Scope        string
}

func parseAuthorizeParams(get func(string) string) (*authorizeParams, error) {
	p := &authorizeParams{
		ClientID:     strings.TrimSpace(get("client_id")),
		RedirectURI:  strings.TrimSpace(get("redirect_uri")),
		ResponseType: strings.TrimSpace(get("response_type")),
		Challenge:    strings.TrimSpace(get("code_challenge")),
		State:        get("state"),
		Scope:        get("scope"),
	}
	if !oauthClients[p.ClientID] {
		return nil, errText("未知的 client_id")
	}
	if err := validateLoopbackRedirect(p.RedirectURI); err != nil {
		return nil, err
	}
	if p.ResponseType != "code" {
		return nil, errText("response_type 只支持 code")
	}
	if get("code_challenge_method") != "S256" {
		return nil, errText("code_challenge_method 只支持 S256")
	}
	if len(p.Challenge) < 43 || len(p.Challenge) > 128 {
		return nil, errText("code_challenge 长度不合法")
	}
	if len(p.State) > 200 {
		return nil, errText("state 过长")
	}
	return p, nil
}

type errText string

func (e errText) Error() string { return string(e) }

func validateLoopbackRedirect(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errText("redirect_uri 不合法")
	}
	if u.Scheme != "http" || u.User != nil || u.Fragment != "" {
		return errText("redirect_uri 必须是本机 http 回调")
	}
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return errText("redirect_uri 只允许 127.0.0.1 / localhost")
	}
	if u.Path != "/callback" {
		return errText("redirect_uri 路径必须是 /callback")
	}
	return nil
}

func (h *OAuthHandler) Authorize(c *gin.Context) {
	p, err := parseAuthorizeParams(c.Query)
	if err != nil {
		h.renderError(c, http.StatusBadRequest, "授权请求无效", err.Error())
		return
	}

	principal := auth.GetPrincipal(c)
	if principal == nil {
		// 登录后回到完整的授权 URL。
		c.Redirect(http.StatusFound, "/login?redirect="+url.QueryEscape(c.Request.URL.RequestURI()))
		return
	}

	scopes := strings.Split(p.Scope, " ")
	if len(scopes) == 0 || strings.TrimSpace(p.Scope) == "" {
		scopes = []string{auth.ScopeRead}
	}
	normalized, nerr := normalizeScopeList(scopes)
	if nerr != nil {
		h.renderError(c, http.StatusBadRequest, "授权请求无效", nerr.Error())
		return
	}

	data := consentData{
		User:          displayName(principal),
		ClientID:      p.ClientID,
		Scopes:        normalized,
		CSRF:          principal.CSRF,
		RedirectURI:   p.RedirectURI,
		State:         p.State,
		CodeChallenge: p.Challenge,
		Scope:         strings.Join(normalized, " "),
	}
	var buf bytes.Buffer
	if err := consentTemplate.Execute(&buf, data); err != nil {
		h.renderError(c, http.StatusInternalServerError, "页面渲染失败", err.Error())
		return
	}
	setHTMLHeaders(c)
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

func (h *OAuthHandler) Approve(c *gin.Context) {
	principal := auth.GetPrincipal(c)
	if principal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		h.renderError(c, http.StatusBadRequest, "请求无效", "表单解析失败")
		return
	}
	if !auth.SafeEqual(c.PostForm("csrf"), principal.CSRF) {
		h.renderError(c, http.StatusForbidden, "请求已过期", "CSRF 校验失败，请重新发起授权")
		return
	}

	p, err := parseAuthorizeParams(c.PostForm)
	if err != nil {
		h.renderError(c, http.StatusBadRequest, "授权请求无效", err.Error())
		return
	}

	if c.PostForm("action") != "approve" {
		redirectWithError(c, p.RedirectURI, p.State, "access_denied", "用户拒绝了授权")
		return
	}

	code, err := h.svc.CreateAuthorizationCode(principal.UserID, p.ClientID, p.RedirectURI, p.Challenge, p.Scope)
	if err != nil {
		redirectWithError(c, p.RedirectURI, p.State, "server_error", "生成授权码失败")
		return
	}

	u, _ := url.Parse(p.RedirectURI)
	q := u.Query()
	q.Set("code", code)
	if p.State != "" {
		q.Set("state", p.State)
	}
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}

func (h *OAuthHandler) Token(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")

	if err := c.Request.ParseForm(); err != nil {
		tokenError(c, http.StatusBadRequest, "invalid_request", "表单解析失败")
		return
	}
	clientID := c.PostForm("client_id")
	if !oauthClients[clientID] {
		tokenError(c, http.StatusUnauthorized, "invalid_client", "未知的 client_id")
		return
	}

	switch c.PostForm("grant_type") {
	case "authorization_code":
		pair, err := h.svc.ExchangeAuthorizationCode(
			c.PostForm("code"), clientID, c.PostForm("redirect_uri"), c.PostForm("code_verifier"),
		)
		if err != nil {
			tokenError(c, http.StatusBadRequest, "invalid_grant", err.Error())
			return
		}
		c.JSON(http.StatusOK, pair)
	case "refresh_token":
		pair, err := h.svc.RefreshAccessToken(c.PostForm("refresh_token"), clientID)
		if err != nil {
			tokenError(c, http.StatusBadRequest, "invalid_grant", err.Error())
			return
		}
		c.JSON(http.StatusOK, pair)
	default:
		tokenError(c, http.StatusBadRequest, "unsupported_grant_type", "只支持 authorization_code 与 refresh_token")
	}
}

func (h *OAuthHandler) Metadata(c *gin.Context) {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	base := scheme + "://" + c.Request.Host
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{auth.ScopeRead, auth.ScopeWrite},
	})
}

// ---------- 辅助 ----------

func normalizeScopeList(raw []string) ([]string, error) {
	seen := map[string]bool{}
	for _, item := range raw {
		for _, s := range strings.FieldsFunc(item, func(r rune) bool { return r == ',' || r == ' ' }) {
			switch s {
			case auth.ScopeRead, auth.ScopeWrite:
				seen[s] = true
			default:
				return nil, errText("不支持的 scope: " + s)
			}
		}
	}
	if seen[auth.ScopeWrite] {
		seen[auth.ScopeRead] = true
	}
	out := make([]string, 0, 2)
	for _, s := range []string{auth.ScopeRead, auth.ScopeWrite} {
		if seen[s] {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		out = []string{auth.ScopeRead}
	}
	return out, nil
}

func displayName(p *auth.Principal) string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	return p.Username
}

func redirectWithError(c *gin.Context, redirectURI, state, code, desc string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "redirect_uri 不合法"})
		return
	}
	q := u.Query()
	q.Set("error", code)
	if desc != "" {
		q.Set("error_description", desc)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}

func tokenError(c *gin.Context, status int, code, desc string) {
	c.AbortWithStatusJSON(status, gin.H{"error": code, "error_description": desc})
}

func setHTMLHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Frame-Options", "DENY")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'")
}

func (h *OAuthHandler) renderError(c *gin.Context, status int, title, detail string) {
	setHTMLHeaders(c)
	c.Data(status, "text/html; charset=utf-8", []byte(errorPage(title, detail)))
}

func errorPage(title, detail string) string {
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>` +
		template.HTMLEscapeString(title) +
		`</title><style>body{font-family:system-ui,-apple-system,"PingFang SC",sans-serif;background:#f6f7f9;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}.card{background:#fff;padding:32px 36px;border-radius:12px;box-shadow:0 8px 30px rgba(0,0,0,.08);max-width:520px}h1{font-size:20px;margin:0 0 12px}p{color:#666;line-height:1.7;margin:0}</style></head><body><div class="card"><h1>` +
		template.HTMLEscapeString(title) +
		`</h1><p>` + template.HTMLEscapeString(detail) + `</p></div></body></html>`
}

type consentData struct {
	User          string
	ClientID      string
	Scopes        []string
	CSRF          string
	RedirectURI   string
	State         string
	CodeChallenge string
	Scope         string
}

func (d consentData) ScopeLabel(s string) string {
	switch s {
	case auth.ScopeRead:
		return "读取学生、课程、成绩与统计数据"
	case auth.ScopeWrite:
		return "新增、修改、删除学生 / 课程 / 成绩数据"
	default:
		return s
	}
}

var consentTemplate = template.Must(template.New("consent").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>授权访问 · 学生管理系统</title>
<style>
  :root { color-scheme: light; }
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
         background:linear-gradient(135deg,#eef4ff 0%,#f7f9fc 55%,#eefaf4 100%);
         font-family:system-ui,-apple-system,"PingFang SC","Microsoft YaHei",sans-serif; color:#1f2937; }
  .card { width:min(440px,92vw); background:#fff; border-radius:16px; padding:32px;
          box-shadow:0 20px 60px rgba(15,23,42,.12); }
  .brand { font-size:13px; letter-spacing:.14em; color:#64748b; text-transform:uppercase; }
  h1 { font-size:22px; margin:10px 0 6px; }
  .sub { color:#64748b; font-size:14px; line-height:1.7; margin:0 0 20px; }
  .client { background:#f8fafc; border:1px solid #e2e8f0; border-radius:10px; padding:12px 14px;
            font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:13px; color:#334155; }
  ul.scopes { list-style:none; padding:0; margin:18px 0 24px; }
  ul.scopes li { display:flex; gap:10px; padding:10px 0; border-bottom:1px dashed #e2e8f0; font-size:14px; }
  ul.scopes li:last-child { border-bottom:0; }
  .dot { width:20px; height:20px; border-radius:50%; background:#dbeafe; color:#1d4ed8;
         display:flex; align-items:center; justify-content:center; font-size:12px; flex:0 0 20px; }
  .actions { display:flex; gap:10px; }
  button { flex:1; border-radius:10px; padding:11px 14px; font-size:14px; cursor:pointer; border:1px solid transparent; }
  .primary { background:#2563eb; color:#fff; }
  .primary:hover { background:#1d4ed8; }
  .ghost { background:#fff; border-color:#cbd5e1; color:#475569; }
  .ghost:hover { background:#f8fafc; }
  .foot { margin-top:16px; font-size:12px; color:#94a3b8; line-height:1.6; }
</style>
</head>
<body>
  <div class="card">
    <div class="brand">Eino OA · Authorization</div>
    <h1>授权访问学生管理系统</h1>
    <p class="sub">已登录为 <strong>{{.User}}</strong>。以下程序请求代表你调用系统 API：</p>
    <div class="client">{{.ClientID}}</div>
    <ul class="scopes">
      {{range .Scopes}}<li><span class="dot">✓</span><span>{{$.ScopeLabel .}}</span></li>{{end}}
    </ul>
    <form method="post" action="/oauth/authorize">
      <input type="hidden" name="csrf" value="{{.CSRF}}">
      <input type="hidden" name="client_id" value="{{.ClientID}}">
      <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
      <input type="hidden" name="response_type" value="code">
      <input type="hidden" name="code_challenge" value="{{.CodeChallenge}}">
      <input type="hidden" name="code_challenge_method" value="S256">
      <input type="hidden" name="scope" value="{{.Scope}}">
      <input type="hidden" name="state" value="{{.State}}">
      <div class="actions">
        <button class="ghost" type="submit" name="action" value="deny">拒绝</button>
        <button class="primary" type="submit" name="action" value="approve">同意授权</button>
      </div>
    </form>
    <p class="foot">授权后程序会获得一枚短期访问令牌。你可以随时在系统的「访问令牌」页面吊销相关令牌。</p>
  </div>
</body>
</html>`))
