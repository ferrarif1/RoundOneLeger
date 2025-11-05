package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ledger/internal/auth"
	"ledger/internal/db"
	"ledger/internal/middleware"
	"ledger/internal/models"
	"ledger/internal/xlsx"
)

// Server wires handlers to the in-memory store and session manager.
type Server struct {
	Database *db.Database
	Store    *models.LedgerStore
	Sessions *auth.Manager
}

// RegisterRoutes attaches handlers to the gin engine.
func (s *Server) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", s.handleHealth)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/request-nonce", s.handleRequestNonce)
		authGroup.POST("/login", s.handleLogin)
		authGroup.POST("/password-login", s.handlePasswordLogin)
		authGroup.POST("/approvals", s.handleSubmitApproval)
	}

	secured := router.Group("/api/v1")
	secured.Use(middleware.RequireSession(s.Sessions))
	{
		secured.GET("/ledgers/:type", s.handleListLedger)
		secured.POST("/ledgers/:type", s.handleCreateLedger)
		secured.PUT("/ledgers/:type/:id", s.handleUpdateLedger)
		secured.DELETE("/ledgers/:type/:id", s.handleDeleteLedger)
		secured.POST("/ledgers/:type/reorder", s.handleReorderLedger)
		secured.POST("/ledgers/:type/import", s.handleImportLedger)
		secured.GET("/ledgers/export", s.handleExportLedger)
		secured.POST("/ledgers/import", s.handleImportWorkbook)
		secured.GET("/ledger-cartesian", s.handleLedgerMatrix)

		secured.GET("/workspaces", s.handleListWorkspaces)
		secured.POST("/workspaces", s.handleCreateWorkspace)
		secured.GET("/workspaces/:id", s.handleGetWorkspace)
		secured.PUT("/workspaces/:id", s.handleUpdateWorkspace)
		secured.DELETE("/workspaces/:id", s.handleDeleteWorkspace)
		secured.POST("/workspaces/:id/import/excel", s.handleImportWorkspaceExcel)
		secured.POST("/workspaces/:id/import/text", s.handleImportWorkspaceText)
		secured.GET("/workspaces/:id/export", s.handleExportWorkspace)

		secured.GET("/users", s.handleListUsers)
		secured.POST("/users", s.handleCreateUser)
		secured.DELETE("/users/:id", s.handleDeleteUser)

		secured.GET("/ip-allowlist", s.handleListAllowlist)
		secured.POST("/ip-allowlist", s.handleCreateAllowlist)
		secured.PUT("/ip-allowlist/:id", s.handleUpdateAllowlist)
		secured.DELETE("/ip-allowlist/:id", s.handleDeleteAllowlist)

		secured.POST("/history/undo", s.handleUndo)
		secured.POST("/history/redo", s.handleRedo)
		secured.GET("/history", s.handleHistoryStatus)

		secured.GET("/audit", s.handleAuditList)
		secured.GET("/audit/verify", s.handleAuditVerify)
		secured.GET("/approvals", s.handleListApprovals)
		secured.POST("/approvals/:id/approve", s.handleApproveRequest)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	payload := gin.H{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339)}
	if s.Database != nil {
		if err := s.Database.PingContext(c.Request.Context()); err != nil {
			payload["database"] = gin.H{"status": "unavailable", "error": err.Error()}
		} else {
			payload["database"] = gin.H{"status": "ok"}
		}
	}
	c.JSON(http.StatusOK, payload)
}

type nonceResponse struct {
	Nonce   string `json:"nonce"`
	Message string `json:"message"`
}

func decodeWebBase64(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, errors.New("empty")
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil {
		return decoded, nil
	}
	return base64.StdEncoding.DecodeString(trimmed)
}

func (s *Server) handleRequestNonce(c *gin.Context) {
	challenge := s.Store.CreateLoginChallenge()
	c.JSON(http.StatusOK, nonceResponse{Nonce: challenge.Nonce, Message: challenge.Message})
}

type jwk struct {
	KTY   string `json:"kty"`
	Curve string `json:"crv"`
	X     string `json:"x"`
	Y     string `json:"y"`
}

type sdidIdentity struct {
	DID          string          `json:"did"`
	Label        string          `json:"label"`
	Roles        []string        `json:"roles"`
	PublicKeyJWK json.RawMessage `json:"publicKeyJwk"`
	Authorized   bool            `json:"authorized"`
}

type sdidProof struct {
	SignatureValue string `json:"signatureValue"`
}

type sdidAuthentication struct {
	CanonicalRequest string          `json:"canonicalRequest"`
	Payload          json.RawMessage `json:"payload"`
}

type sdidLoginResponse struct {
	Identity       sdidIdentity        `json:"identity"`
	Challenge      string              `json:"challenge"`
	Signature      string              `json:"signature"`
	Proof          sdidProof           `json:"proof"`
	Authentication *sdidAuthentication `json:"authentication"`
	Authorized     bool                `json:"authorized"`
}

type approvalResponse struct {
	ID               string     `json:"id"`
	ApplicantDid     string     `json:"applicantDid"`
	ApplicantLabel   string     `json:"applicantLabel"`
	ApplicantRoles   []string   `json:"applicantRoles"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	ApprovedAt       *time.Time `json:"approvedAt,omitempty"`
	ApproverDid      string     `json:"approverDid,omitempty"`
	ApproverLabel    string     `json:"approverLabel,omitempty"`
	ApproverRoles    []string   `json:"approverRoles,omitempty"`
	SigningChallenge string     `json:"signingChallenge,omitempty"`
}

type approvalClaims struct {
	Approved   bool   `json:"approved"`
	Approver   string `json:"approver"`
	ApproverID string `json:"approverDid"`
	ApprovedAt string `json:"approvedAt"`
}

type resourcesClaims struct {
	Roles         []string       `json:"roles"`
	Approved      bool           `json:"approved"`
	Certification approvalClaims `json:"certification"`
}

type authenticationClaims struct {
	Issuer        string          `json:"iss"`
	Subject       string          `json:"sub"`
	Audience      string          `json:"aud"`
	Purpose       string          `json:"purpose"`
	Statement     string          `json:"statement"`
	RequestID     string          `json:"requestId"`
	Nonce         string          `json:"nonce"`
	Resources     resourcesClaims `json:"resources"`
	Approved      bool            `json:"approved"`
	Certification approvalClaims  `json:"certification"`
}

type loginRequest struct {
	Nonce    string            `json:"nonce"`
	Response sdidLoginResponse `json:"response"`
}

type passwordLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	if strings.TrimSpace(req.Nonce) == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	challenge, err := s.resolveChallenge(req.Nonce, &req.Response)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	claims, _, err := s.verifySdidLoginResponse(challenge, &req.Response)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	approvalStatus, approval, evalErr := s.evaluateIdentityApproval(&req.Response, claims)
	if evalErr != nil {
		status := http.StatusUnauthorized
		if errors.Is(evalErr, models.ErrIdentityNotApproved) {
			status = http.StatusForbidden
		}
		payload := gin.H{"error": evalErr.Error(), "status": approvalStatus}
		if approval != nil {
			payload["approval"] = approvalToResponse(approval)
		}
		c.AbortWithStatusJSON(status, payload)
		return
	}
	sdid := strings.TrimSpace(req.Response.Identity.DID)
	s.Store.RecordLogin(sdid)
	session, err := s.Sessions.Issue(sdid, sdid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session_issue_failed"})
		return
	}
	c.JSON(http.StatusOK, session)
}

func (s *Server) handlePasswordLogin(c *gin.Context) {
	var req passwordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	user, err := s.Store.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, models.ErrUsernameInvalid) || errors.Is(err, models.ErrPasswordTooShort) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	session, err := s.Sessions.Issue(user.Username, user.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session_issue_failed"})
		return
	}
	s.Store.RecordLogin(user.Username)
	c.JSON(http.StatusOK, gin.H{
		"token":     session.Token,
		"username":  user.Username,
		"admin":     user.Admin,
		"issuedAt":  session.IssuedAt,
		"expiresAt": session.ExpiresAt,
	})
}

func (s *Server) resolveChallenge(nonce string, resp *sdidLoginResponse) (*models.LoginChallenge, error) {
	challenge, err := s.Store.ConsumeLoginChallenge(nonce)
	if err == nil {
		return challenge, nil
	}
	if !errors.Is(err, models.ErrLoginChallengeNotFound) {
		return nil, err
	}
	if resp == nil {
		return nil, models.ErrLoginChallengeNotFound
	}
	if strings.TrimSpace(resp.Challenge) != strings.TrimSpace(nonce) {
		return nil, models.ErrLoginChallengeNotFound
	}
	fallback := &models.LoginChallenge{
		Nonce:     strings.TrimSpace(nonce),
		Message:   strings.TrimSpace(resp.Challenge),
		CreatedAt: time.Now().UTC(),
	}
	if fallback.Message == "" {
		fallback.Message = fallback.Nonce
	}
	return fallback, nil
}

func (s *Server) handleSubmitApproval(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	if strings.TrimSpace(req.Nonce) == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	challenge, err := s.resolveChallenge(req.Nonce, &req.Response)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	claims, signed, err := s.verifySdidLoginResponse(challenge, &req.Response)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	status, approval, evalErr := s.evaluateIdentityApproval(&req.Response, claims)
	roles := mergeRoles(normaliseRoles(req.Response.Identity.Roles), normaliseRoles(claims.Resources.Roles))
	signatureValue := strings.TrimSpace(req.Response.Proof.SignatureValue)
	if signatureValue == "" {
		signatureValue = strings.TrimSpace(req.Response.Signature)
	}
	if evalErr == nil {
		payload := gin.H{"status": status}
		if approval != nil {
			view := approvalToResponse(approval)
			payload["approval"] = view
		}
		c.JSON(http.StatusOK, payload)
		return
	}
	if !errors.Is(evalErr, models.ErrIdentityNotApproved) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": evalErr.Error()})
		return
	}
	record, submitErr := s.Store.SubmitApproval(req.Response.Identity.DID, req.Response.Identity.Label, roles, challenge.Message, signatureValue, signed)
	if submitErr != nil {
		if errors.Is(submitErr, models.ErrApprovalAlreadyPending) || errors.Is(submitErr, models.ErrApprovalAlreadyCompleted) {
			view := approvalToResponse(record)
			c.JSON(http.StatusOK, gin.H{"status": view.Status, "approval": view})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": submitErr.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": record.Status, "approval": approvalToResponse(record)})
}

type approveRequest struct {
	Response sdidLoginResponse `json:"response"`
}

func (s *Server) handleListApprovals(c *gin.Context) {
	actor := currentSession(c, s.Sessions)
	if !s.Store.IdentityIsAdmin(actor) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	approvals := s.Store.ListApprovals()
	views := make([]approvalResponse, 0, len(approvals))
	for _, item := range approvals {
		views = append(views, approvalToResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": views})
}

func (s *Server) handleApproveRequest(c *gin.Context) {
	actor := currentSession(c, s.Sessions)
	if !s.Store.IdentityIsAdmin(actor) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	approval, err := s.Store.ApprovalByID(id)
	if err != nil {
		status := http.StatusNotFound
		if !errors.Is(err, models.ErrApprovalNotFound) {
			status = http.StatusInternalServerError
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	if approval.Status != models.ApprovalStatusPending {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "approval_not_pending"})
		return
	}
	var req approveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	expectedChallenge := buildApprovalSigningChallenge(approval)
	pseudoChallenge := &models.LoginChallenge{Nonce: expectedChallenge, Message: expectedChallenge, CreatedAt: time.Now().UTC()}
	claims, _, err := s.verifySdidLoginResponse(pseudoChallenge, &req.Response)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	approverDID := strings.TrimSpace(req.Response.Identity.DID)
	if approverDID == "" || approverDID != actor {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin_mismatch"})
		return
	}
	if !s.Store.IdentityIsAdmin(approverDID) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin_required"})
		return
	}
	_, _, evalErr := s.evaluateIdentityApproval(&req.Response, claims)
	if evalErr != nil && !errors.Is(evalErr, models.ErrIdentityNotApproved) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": evalErr.Error()})
		return
	}
	signatureValue := strings.TrimSpace(req.Response.Proof.SignatureValue)
	if signatureValue == "" {
		signatureValue = strings.TrimSpace(req.Response.Signature)
	}
	approverProfile := s.Store.IdentityProfileByDID(approverDID)
	record, approveErr := s.Store.ApproveRequest(id, approverProfile, expectedChallenge, signatureValue)
	if approveErr != nil {
		status := http.StatusInternalServerError
		if errors.Is(approveErr, models.ErrApprovalAlreadyCompleted) {
			status = http.StatusConflict
		}
		c.AbortWithStatusJSON(status, gin.H{"error": approveErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": record.Status, "approval": approvalToResponse(record)})
}

func canonicalizeJSONValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		b, _ := json.Marshal(v)
		return string(b)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = canonicalizeJSONValue(item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, strconv.Quote(key)+":"+canonicalizeJSONValue(v[key]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func canonicalizeJSON(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	return canonicalizeJSONValue(value), nil
}

func decodeJWK(data json.RawMessage) (*ecdsa.PublicKey, error) {
	if len(data) == 0 {
		return nil, errors.New("missing_public_key")
	}
	var key jwk
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, errors.New("invalid_public_key")
	}
	if !strings.EqualFold(key.KTY, "EC") {
		return nil, errors.New("unsupported_public_key")
	}
	if key.Curve != "P-256" && key.Curve != "secp256r1" {
		return nil, errors.New("unsupported_curve")
	}
	decodeCoordinate := func(value string) (*big.Int, error) {
		bytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		return new(big.Int).SetBytes(bytes), nil
	}
	x, err := decodeCoordinate(key.X)
	if err != nil {
		return nil, errors.New("invalid_public_key")
	}
	y, err := decodeCoordinate(key.Y)
	if err != nil {
		return nil, errors.New("invalid_public_key")
	}
	curve := elliptic.P256()
	if !curve.IsOnCurve(x, y) {
		return nil, errors.New("invalid_public_key")
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func normaliseRoles(roles []string) []string {
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func mergeRoles(sets ...[]string) []string {
	total := 0
	for _, set := range sets {
		total += len(set)
	}
	merged := make([]string, 0, total)
	for _, set := range sets {
		if len(set) == 0 {
			continue
		}
		merged = append(merged, set...)
	}
	return merged
}

func containsAdminRole(roles []string) bool {
	for _, role := range roles {
		lowered := strings.ToLower(role)
		if lowered == "" {
			continue
		}
		if strings.Contains(lowered, "admin") || lowered == "approver" || strings.Contains(lowered, "approver") {
			return true
		}
	}
	return false
}

func claimsApproved(claims authenticationClaims) bool {
	if claims.Approved || claims.Certification.Approved {
		return true
	}
	if claims.Resources.Approved || claims.Resources.Certification.Approved {
		return true
	}
	if claims.Certification.Approver != "" || claims.Certification.ApproverID != "" {
		return true
	}
	if claims.Resources.Certification.Approver != "" || claims.Resources.Certification.ApproverID != "" {
		return true
	}
	if claims.Certification.ApprovedAt != "" || claims.Resources.Certification.ApprovedAt != "" {
		return true
	}
	return false
}

func (s *Server) verifySdidLoginResponse(challenge *models.LoginChallenge, resp *sdidLoginResponse) (authenticationClaims, string, error) {
	if resp == nil {
		return authenticationClaims{}, "", errors.New("invalid_payload")
	}
	sdid := strings.TrimSpace(resp.Identity.DID)
	if sdid == "" {
		return authenticationClaims{}, "", errors.New("missing_sdid")
	}
	expectedChallenge := ""
	challengeMessage := ""
	if challenge != nil {
		expectedChallenge = strings.TrimSpace(challenge.Nonce)
		challengeMessage = strings.TrimSpace(challenge.Message)
	}
	providedChallenge := strings.TrimSpace(resp.Challenge)
	signatureValue := strings.TrimSpace(resp.Proof.SignatureValue)
	if signatureValue == "" {
		signatureValue = strings.TrimSpace(resp.Signature)
	}
	if signatureValue == "" {
		return authenticationClaims{}, "", errors.New("missing_signature")
	}
	sigBytes, err := decodeWebBase64(signatureValue)
	if err != nil {
		return authenticationClaims{}, "", errors.New("invalid_signature")
	}
	publicKey, err := decodeJWK(resp.Identity.PublicKeyJWK)
	if err != nil {
		return authenticationClaims{}, "", err
	}
	signedData := ""
	var claims authenticationClaims
	if resp.Authentication != nil {
		signedData = strings.TrimSpace(resp.Authentication.CanonicalRequest)
		if len(resp.Authentication.Payload) > 0 {
			_ = json.Unmarshal(resp.Authentication.Payload, &claims)
			canonical, err := canonicalizeJSON(resp.Authentication.Payload)
			if err != nil {
				return authenticationClaims{}, "", errors.New("invalid_authentication_payload")
			}
			if signedData != "" && canonical != signedData {
				return authenticationClaims{}, "", errors.New("authentication_mismatch")
			}
			if signedData == "" {
				signedData = canonical
			}
		}
	}
	if signedData == "" {
		signedData = providedChallenge
	}
	if signedData == "" {
		signedData = challengeMessage
	}
	if signedData == "" {
		return authenticationClaims{}, "", errors.New("missing_challenge")
	}
	hash := sha256.Sum256([]byte(signedData))
	verified := ecdsa.VerifyASN1(publicKey, hash[:], sigBytes)
	raw := sigBytes
	if len(raw) == 65 && raw[0] == 0x00 {
		raw = raw[1:]
	}
	if !verified && len(raw) == 64 {
		r := new(big.Int).SetBytes(raw[:32])
		sVal := new(big.Int).SetBytes(raw[32:])
		if ecdsa.Verify(publicKey, hash[:], r, sVal) {
			verified = true
		}
	}
	if !verified {
		return authenticationClaims{}, "", models.ErrSignatureInvalid
	}
	if !challengeSatisfied(expectedChallenge, providedChallenge, signedData, claims.Nonce, challengeMessage) {
		return authenticationClaims{}, "", errors.New("challenge_mismatch")
	}
	return claims, signedData, nil
}

func (s *Server) evaluateIdentityApproval(resp *sdidLoginResponse, claims authenticationClaims) (string, *models.IdentityApproval, error) {
	if resp == nil {
		return models.ApprovalStatusMissing, nil, errors.New("invalid_payload")
	}
	sdid := strings.TrimSpace(resp.Identity.DID)
	if sdid == "" {
		return models.ApprovalStatusMissing, nil, errors.New("missing_sdid")
	}
	roles := normaliseRoles(resp.Identity.Roles)
	resourceRoles := normaliseRoles(claims.Resources.Roles)
	mergedRoles := mergeRoles(roles, resourceRoles)
	admin := containsAdminRole(mergedRoles)
	approvedByClaims := resp.Identity.Authorized || resp.Authorized || claimsApproved(claims)
	profile := s.Store.UpdateIdentityProfile(sdid, resp.Identity.Label, mergedRoles, admin, approvedByClaims)
	if profile.Admin || profile.Approved {
		return models.ApprovalStatusApproved, nil, nil
	}
	status, approval := s.Store.IdentityApprovalState(sdid)
	if status == models.ApprovalStatusApproved {
		s.Store.UpdateIdentityProfile(sdid, resp.Identity.Label, mergedRoles, profile.Admin, true)
		return status, approval, nil
	}
	return status, approval, models.ErrIdentityNotApproved
}

func approvalToResponse(approval *models.IdentityApproval) approvalResponse {
	if approval == nil {
		return approvalResponse{}
	}
	resp := approvalResponse{
		ID:             approval.ID,
		ApplicantDid:   approval.ApplicantDid,
		ApplicantLabel: approval.ApplicantLabel,
		ApplicantRoles: append([]string{}, approval.ApplicantRoles...),
		Status:         approval.Status,
		CreatedAt:      approval.CreatedAt,
	}
	if approval.ApprovedAt != nil {
		approved := *approval.ApprovedAt
		resp.ApprovedAt = &approved
	}
	if approval.ApproverDid != "" {
		resp.ApproverDid = approval.ApproverDid
	}
	if approval.ApproverLabel != "" {
		resp.ApproverLabel = approval.ApproverLabel
	}
	if len(approval.ApproverRoles) > 0 {
		resp.ApproverRoles = append([]string{}, approval.ApproverRoles...)
	}
	if approval.Status == models.ApprovalStatusPending {
		resp.SigningChallenge = buildApprovalSigningChallenge(approval)
	}
	return resp
}

func buildApprovalSigningChallenge(approval *models.IdentityApproval) string {
	if approval == nil {
		return ""
	}
	payload := map[string]any{
		"type":           "roundone:identity-approval",
		"approvalId":     approval.ID,
		"applicantDid":   approval.ApplicantDid,
		"applicantLabel": approval.ApplicantLabel,
		"applicantRoles": approval.ApplicantRoles,
		"submittedAt":    approval.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	return canonicalizeJSONValue(payload)
}

func challengeSatisfied(expected, provided, signedData, claimsNonce, challengeMessage string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	candidates := []string{
		strings.TrimSpace(provided),
		strings.TrimSpace(signedData),
		strings.TrimSpace(claimsNonce),
		strings.TrimSpace(challengeMessage),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if candidate == expected {
			return true
		}
	}
	return false
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		parts := strings.Split(v, ",")
		candidate := strings.TrimSpace(parts[0])
		if ip := net.ParseIP(candidate); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if ip := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); ip != nil {
			return ip.String()
		}
		return r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}

func (s *Server) handleListLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	entries := s.Store.ListEntries(typ)
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

type ledgerRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Attributes  map[string]string   `json:"attributes"`
	Tags        []string            `json:"tags"`
	Links       map[string][]string `json:"links"`
}

type workspaceColumnPayload struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Width int    `json:"width,omitempty"`
}

type workspaceRowPayload struct {
	ID        string            `json:"id"`
	Cells     map[string]string `json:"cells"`
	CreatedAt time.Time         `json:"createdAt,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt,omitempty"`
}

type workspaceResponse struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	Kind      string                   `json:"kind"`
	ParentID  string                   `json:"parentId,omitempty"`
	Columns   []workspaceColumnPayload `json:"columns"`
	Rows      []workspaceRowPayload    `json:"rows"`
	Document  string                   `json:"document,omitempty"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
}

type workspaceTreeItem struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Kind      string              `json:"kind"`
	ParentID  string              `json:"parentId,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Children  []workspaceTreeItem `json:"children,omitempty"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Admin     bool      `json:"admin"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type userCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

type workspaceRequest struct {
	Name     string                   `json:"name"`
	Document string                   `json:"document"`
	Columns  []workspaceColumnPayload `json:"columns"`
	Rows     []workspaceRowPayload    `json:"rows"`
	Kind     string                   `json:"kind"`
	ParentID string                   `json:"parentId"`
}

type workspaceUpdateRequest struct {
	Name     *string                   `json:"name,omitempty"`
	Document *string                   `json:"document,omitempty"`
	Columns  *[]workspaceColumnPayload `json:"columns,omitempty"`
	Rows     *[]workspaceRowPayload    `json:"rows,omitempty"`
	ParentID *string                   `json:"parentId,omitempty"`
}

type workspaceTextImportRequest struct {
	Text      string `json:"text"`
	Delimiter string `json:"delimiter"`
	HasHeader *bool  `json:"hasHeader"`
}

func (s *Server) handleCreateLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	var req ledgerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	entry := models.LedgerEntry{
		Name:        req.Name,
		Description: req.Description,
		Attributes:  req.Attributes,
		Tags:        req.Tags,
		Links:       convertLinks(req.Links),
	}
	session := currentSession(c, s.Sessions)
	created, err := s.Store.CreateEntry(typ, entry, session)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, created)
}

func convertLinks(input map[string][]string) map[models.LedgerType][]string {
	if input == nil {
		return nil
	}
	out := make(map[models.LedgerType][]string, len(input))
	for key, values := range input {
		if typ, ok := parseLedgerType(key); ok {
			out[typ] = append([]string{}, values...)
		}
	}
	return out
}

func (s *Server) handleUpdateLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	var req ledgerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	session := currentSession(c, s.Sessions)
	updated, err := s.Store.UpdateEntry(typ, c.Param("id"), models.LedgerEntry{
		Name:        req.Name,
		Description: req.Description,
		Attributes:  req.Attributes,
		Tags:        req.Tags,
		Links:       convertLinks(req.Links),
	}, session)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) handleDeleteLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	session := currentSession(c, s.Sessions)
	if err := s.Store.DeleteEntry(typ, c.Param("id"), session); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type reorderRequest struct {
	IDs []string `json:"ids"`
}

func (s *Server) handleReorderLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	session := currentSession(c, s.Sessions)
	entries, err := s.Store.ReorderEntries(typ, req.IDs, session)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

type importRequest struct {
	Data string `json:"data"`
}

func (s *Server) handleImportLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	var req importRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	raw, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_base64"})
		return
	}
	workbook, err := xlsx.Decode(raw)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_workbook"})
		return
	}
	sheetName := sheetNameForType(typ)
	sheet, ok := workbook.SheetByName(sheetName)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "sheet_missing"})
		return
	}
	entries := parseLedgerSheet(typ, sheet)
	session := currentSession(c, s.Sessions)
	s.Store.ReplaceEntries(typ, entries, session)
	c.JSON(http.StatusOK, gin.H{"items": s.Store.ListEntries(typ)})
}

var ipRegex = regexp.MustCompile(`(?i)^(?:\d{1,3}\.){3}\d{1,3}$|^(?:[a-f0-9]{0,4}:){2,7}[a-f0-9]{0,4}$`)

func parseLedgerSheet(typ models.LedgerType, sheet xlsx.Sheet) []models.LedgerEntry {
	if len(sheet.Rows) == 0 {
		return nil
	}
	headers := normaliseHeaders(sheet.Rows[0])
	entries := make([]models.LedgerEntry, 0, len(sheet.Rows)-1)
	for _, row := range sheet.Rows[1:] {
		if len(row) == 0 {
			continue
		}
		entry := models.LedgerEntry{Attributes: map[string]string{}, Links: map[models.LedgerType][]string{}}
		for idx, header := range headers {
			if idx >= len(row) {
				continue
			}
			value := strings.TrimSpace(row[idx])
			switch header {
			case "id":
				entry.ID = value
			case "name":
				entry.Name = value
			case "description":
				entry.Description = value
			case "tags":
				entry.Tags = strings.Split(value, ";")
			default:
				if header == "" {
					if typ == models.LedgerTypeIP && ipRegex.MatchString(value) {
						entry.Attributes["address"] = value
						if entry.Name == "" {
							entry.Name = value
						}
					}
					continue
				}
				if strings.HasPrefix(header, "link_") {
					if typ, ok := parseLedgerType(strings.TrimPrefix(header, "link_")); ok {
						entry.Links[typ] = strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == ',' })
					}
					continue
				}
				if entry.Attributes == nil {
					entry.Attributes = make(map[string]string)
				}
				entry.Attributes[header] = value
			}
		}
		if typ == models.LedgerTypeIP && entry.Name == "" {
			if addr := entry.Attributes["address"]; addr != "" {
				entry.Name = addr
			}
		}
		if entry.ID == "" {
			entry.ID = models.GenerateID(string(typ))
		}
		entry.CreatedAt = time.Now().UTC()
		entries = append(entries, entry)
	}
	return entries
}

func normaliseHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, header := range headers {
		h := strings.TrimSpace(strings.ToLower(header))
		out[i] = h
	}
	return out
}

func (s *Server) handleExportLedger(c *gin.Context) {
	workbook := s.buildWorkbook()
	data, err := xlsx.Encode(workbook)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "export_failed"})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=ledger.xlsx")
	_, _ = c.Writer.Write(data)
}

func (s *Server) handleImportWorkbook(c *gin.Context) {
	var req importRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	raw, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_base64"})
		return
	}
	workbook, err := xlsx.Decode(raw)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_workbook"})
		return
	}
	session := currentSession(c, s.Sessions)
	for _, typ := range models.AllLedgerTypes {
		sheetName := sheetNameForType(typ)
		if sheet, ok := workbook.SheetByName(sheetName); ok {
			entries := parseLedgerSheet(typ, sheet)
			s.Store.ReplaceEntries(typ, entries, session)
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "imported"})
}

func (s *Server) handleLedgerMatrix(c *gin.Context) {
	matrix := s.buildCartesianRows()
	c.JSON(http.StatusOK, gin.H{"rows": matrix})
}

func (s *Server) handleListWorkspaces(c *gin.Context) {
	items := s.Store.ListWorkspaces()
	buckets := make(map[string][]workspaceTreeItem)
	for _, item := range items {
		parent := strings.TrimSpace(item.ParentID)
		buckets[parent] = append(buckets[parent], workspaceTreeItemFromModel(item))
	}
	var build func(parent string) []workspaceTreeItem
	build = func(parent string) []workspaceTreeItem {
		nodes := buckets[parent]
		result := make([]workspaceTreeItem, len(nodes))
		for i, node := range nodes {
			node.Children = build(node.ID)
			result[i] = node
		}
		return result
	}
	c.JSON(http.StatusOK, gin.H{"items": build("")})
}

func (s *Server) handleCreateWorkspace(c *gin.Context) {
	var req workspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	actor := currentSession(c, s.Sessions)
	kind := models.ParseWorkspaceKind(req.Kind)
	workspace, err := s.Store.CreateWorkspace(
		req.Name,
		kind,
		req.ParentID,
		payloadColumnsToModel(req.Columns),
		payloadRowsToModel(req.Rows),
		req.Document,
		actor,
	)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceParentInvalid) || errors.Is(err, models.ErrWorkspaceKindUnsupported) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleGetWorkspace(c *gin.Context) {
	workspace, err := s.Store.GetWorkspace(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleUpdateWorkspace(c *gin.Context) {
	var req workspaceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	update := models.WorkspaceUpdate{}
	if req.Name != nil {
		update.SetName = true
		update.Name = *req.Name
	}
	if req.Document != nil {
		update.SetDocument = true
		update.Document = *req.Document
	}
	if req.Columns != nil {
		update.SetColumns = true
		update.Columns = payloadColumnsToModel(*req.Columns)
	}
	if req.Rows != nil {
		update.SetRows = true
		update.Rows = payloadRowsToModel(*req.Rows)
	}
	if req.ParentID != nil {
		update.SetParent = true
		update.ParentID = strings.TrimSpace(*req.ParentID)
	}
	actor := currentSession(c, s.Sessions)
	workspace, err := s.Store.UpdateWorkspace(c.Param("id"), update, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, models.ErrWorkspaceParentInvalid) || errors.Is(err, models.ErrWorkspaceKindUnsupported) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleDeleteWorkspace(c *gin.Context) {
	if err := s.Store.DeleteWorkspace(c.Param("id"), currentSession(c, s.Sessions)); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleImportWorkspaceExcel(c *gin.Context) {
	uploaded, _, err := c.Request.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing_file"})
		return
	}
	defer uploaded.Close()

	data, err := io.ReadAll(uploaded)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "file_read_failed"})
		return
	}

	workbook, err := xlsx.Decode(data)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_workbook"})
		return
	}
	if len(workbook.Sheets) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "workbook_empty"})
		return
	}
	sheet := workbook.Sheets[0]
	headers := []string{}
	rows := [][]string{}
	if len(sheet.Rows) > 0 {
		headers = append(headers, sheet.Rows[0]...)
		for _, row := range sheet.Rows[1:] {
			copied := append([]string{}, row...)
			rows = append(rows, copied)
		}
	}
	actor := currentSession(c, s.Sessions)
	workspace, err := s.Store.ReplaceWorkspaceData(c.Param("id"), headers, rows, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, models.ErrWorkspaceKindUnsupported) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleImportWorkspaceText(c *gin.Context) {
	var req workspaceTextImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "empty_text"})
		return
	}
	delimiter := normaliseDelimiter(req.Delimiter)
	hasHeader := true
	if req.HasHeader != nil {
		hasHeader = *req.HasHeader
	}
	headers, records := parseDelimitedText(text, delimiter, hasHeader)
	actor := currentSession(c, s.Sessions)
	workspace, err := s.Store.ReplaceWorkspaceData(c.Param("id"), headers, records, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, models.ErrWorkspaceKindUnsupported) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleExportWorkspace(c *gin.Context) {
	workspace, err := s.Store.GetWorkspace(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}

	if !models.WorkspaceKindSupportsTable(workspace.Kind) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": models.ErrWorkspaceKindUnsupported.Error()})
		return
	}

	rows := make([][]string, 0, len(workspace.Rows)+1)
	if len(workspace.Columns) > 0 {
		header := make([]string, len(workspace.Columns))
		for i, column := range workspace.Columns {
			header[i] = column.Title
		}
		rows = append(rows, header)
	}
	for _, row := range workspace.Rows {
		record := make([]string, len(workspace.Columns))
		for i, column := range workspace.Columns {
			record[i] = row.Cells[column.ID]
		}
		rows = append(rows, record)
	}
	sheetName := workspace.Name
	if strings.TrimSpace(sheetName) == "" {
		sheetName = "workspace"
	}
	workbook := xlsx.Workbook{Sheets: []xlsx.Sheet{{Name: sheetName, Rows: rows}}}
	encoded, err := xlsx.Encode(workbook)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "encode_failed"})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.xlsx\"", "workspace"))
	c.Writer.Write(encoded)
}

func (s *Server) handleListUsers(c *gin.Context) {
	session := currentSession(c, s.Sessions)
	if !s.Store.IsUserAdmin(session) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin_required"})
		return
	}
	users := s.Store.ListUsers()
	items := make([]userResponse, 0, len(users))
	for _, user := range users {
		items = append(items, userToResponse(user))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) handleCreateUser(c *gin.Context) {
	session := currentSession(c, s.Sessions)
	if !s.Store.IsUserAdmin(session) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin_required"})
		return
	}
	var req userCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	user, err := s.Store.CreateUser(req.Username, req.Password, req.Admin, session)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, models.ErrUsernameInvalid), errors.Is(err, models.ErrPasswordTooShort):
			status = http.StatusBadRequest
		case errors.Is(err, models.ErrUserExists):
			status = http.StatusConflict
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": userToResponse(user)})
}

func (s *Server) handleDeleteUser(c *gin.Context) {
	session := currentSession(c, s.Sessions)
	if !s.Store.IsUserAdmin(session) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin_required"})
		return
	}
	if err := s.Store.DeleteUser(c.Param("id"), session); err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, models.ErrUserNotFound):
			status = http.StatusNotFound
		case errors.Is(err, models.ErrUserDeleteLastAdmin):
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func workspaceToResponse(workspace *models.Workspace) workspaceResponse {
	if workspace == nil {
		return workspaceResponse{}
	}
	columns := make([]workspaceColumnPayload, len(workspace.Columns))
	for i, column := range workspace.Columns {
		columns[i] = workspaceColumnPayload{ID: column.ID, Title: column.Title, Width: column.Width}
	}
	rows := make([]workspaceRowPayload, len(workspace.Rows))
	for i, row := range workspace.Rows {
		cells := make(map[string]string, len(row.Cells))
		for key, value := range row.Cells {
			cells[key] = value
		}
		rows[i] = workspaceRowPayload{ID: row.ID, Cells: cells, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	}
	return workspaceResponse{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Kind:      string(models.NormalizeWorkspaceKind(workspace.Kind)),
		ParentID:  strings.TrimSpace(workspace.ParentID),
		Columns:   columns,
		Rows:      rows,
		Document:  workspace.Document,
		CreatedAt: workspace.CreatedAt,
		UpdatedAt: workspace.UpdatedAt,
	}
}

func workspaceTreeItemFromModel(workspace *models.Workspace) workspaceTreeItem {
	if workspace == nil {
		return workspaceTreeItem{}
	}
	return workspaceTreeItem{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Kind:      string(models.NormalizeWorkspaceKind(workspace.Kind)),
		ParentID:  strings.TrimSpace(workspace.ParentID),
		CreatedAt: workspace.CreatedAt,
		UpdatedAt: workspace.UpdatedAt,
	}
}

func userToResponse(user *models.User) userResponse {
	if user == nil {
		return userResponse{}
	}
	return userResponse{
		ID:        user.ID,
		Username:  user.Username,
		Admin:     user.Admin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func payloadColumnsToModel(columns []workspaceColumnPayload) []models.WorkspaceColumn {
	if len(columns) == 0 {
		return nil
	}
	out := make([]models.WorkspaceColumn, 0, len(columns))
	for _, column := range columns {
		out = append(out, models.WorkspaceColumn{ID: strings.TrimSpace(column.ID), Title: column.Title, Width: column.Width})
	}
	return out
}

func payloadRowsToModel(rows []workspaceRowPayload) []models.WorkspaceRow {
	if len(rows) == 0 {
		return nil
	}
	out := make([]models.WorkspaceRow, 0, len(rows))
	for _, row := range rows {
		cells := make(map[string]string, len(row.Cells))
		for key, value := range row.Cells {
			cells[key] = value
		}
		out = append(out, models.WorkspaceRow{ID: strings.TrimSpace(row.ID), Cells: cells, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return out
}

func normaliseDelimiter(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "", "tab", "\\t":
		return "\t"
	case "comma", ",":
		return ","
	case "semicolon", ";":
		return ";"
	case "space":
		return " "
	default:
		return trimmed
	}
}

func parseDelimitedText(text string, delimiter string, hasHeader bool) ([]string, [][]string) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	if len(cleaned) == 0 {
		return []string{}, [][]string{}
	}
	header := []string{}
	data := [][]string{}
	for idx, line := range cleaned {
		fields := splitWithDelimiter(line, delimiter)
		if idx == 0 && hasHeader {
			header = append(header, fields...)
			continue
		}
		data = append(data, fields)
	}
	return header, data
}

func splitWithDelimiter(line, delimiter string) []string {
	if delimiter == "" {
		return strings.Fields(line)
	}
	parts := strings.Split(line, delimiter)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

func (s *Server) handleListAllowlist(c *gin.Context) {
	entries := s.Store.ListAllowlist()
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

type allowRequest struct {
	Label       string `json:"label"`
	CIDR        string `json:"cidr"`
	Description string `json:"description"`
}

func (s *Server) handleCreateAllowlist(c *gin.Context) {
	var req allowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	entry, err := s.Store.AppendAllowlist(&models.IPAllowlistEntry{Label: req.Label, CIDR: req.CIDR, Description: req.Description}, currentSession(c, s.Sessions))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (s *Server) handleUpdateAllowlist(c *gin.Context) {
	var req allowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	entry := &models.IPAllowlistEntry{ID: c.Param("id"), Label: req.Label, CIDR: req.CIDR, Description: req.Description}
	updated, err := s.Store.AppendAllowlist(entry, currentSession(c, s.Sessions))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) handleDeleteAllowlist(c *gin.Context) {
	if !s.Store.RemoveAllowlist(c.Param("id"), currentSession(c, s.Sessions)) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleUndo(c *gin.Context) {
	if err := s.Store.Undo(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleRedo(c *gin.Context) {
	if err := s.Store.Redo(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleHistoryStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"undo": s.Store.CanUndo(), "redo": s.Store.CanRedo()})
}

func (s *Server) handleAuditList(c *gin.Context) {
	audits := s.Store.ListAudits()
	c.JSON(http.StatusOK, gin.H{"items": audits})
}

func (s *Server) handleAuditVerify(c *gin.Context) {
	ok := s.Store.VerifyAuditChain()
	c.JSON(http.StatusOK, gin.H{"verified": ok})
}

func parseLedgerType(value string) (models.LedgerType, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "ips", "ip":
		return models.LedgerTypeIP, true
	case "personnel", "people", "person":
		return models.LedgerTypePersonnel, true
	case "systems", "system":
		return models.LedgerTypeSystem, true
	default:
		return "", false
	}
}

func currentSession(c *gin.Context, manager *auth.Manager) string {
	if c == nil {
		return "system"
	}
	if sessionAny, ok := c.Get(middleware.ContextSessionKey); ok {
		if session, ok := sessionAny.(*auth.Session); ok {
			return session.Username
		}
	}
	return "system"
}

func (s *Server) buildWorkbook() xlsx.Workbook {
	sheets := make([]xlsx.Sheet, 0, len(models.AllLedgerTypes)+1)
	for _, typ := range models.AllLedgerTypes {
		entries := s.Store.ListEntries(typ)
		sheet := xlsx.Sheet{Name: sheetNameForType(typ)}
		header := []string{"ID", "Name", "Description", "Tags"}
		keys := attributeKeys(entries)
		header = append(header, keys...)
		for _, other := range models.AllLedgerTypes {
			if other == typ {
				continue
			}
			header = append(header, "link_"+string(other))
		}
		sheet.Rows = append(sheet.Rows, header)
		for _, entry := range entries {
			row := []string{entry.ID, entry.Name, entry.Description, strings.Join(entry.Tags, ";")}
			for _, key := range keys {
				row = append(row, entry.Attributes[key])
			}
			for _, other := range models.AllLedgerTypes {
				if other == typ {
					continue
				}
				row = append(row, strings.Join(entry.Links[other], ";"))
			}
			sheet.Rows = append(sheet.Rows, row)
		}
		sheets = append(sheets, sheet)
	}
	combined := s.buildCartesianRows()
	matrixSheet := xlsx.Sheet{Name: "Matrix"}
	matrixHeader := []string{"IP", "Personnel", "System"}
	matrixSheet.Rows = append(matrixSheet.Rows, matrixHeader)
	matrixSheet.Rows = append(matrixSheet.Rows, combined...)
	sheets = append(sheets, matrixSheet)
	workbook := xlsx.Workbook{Sheets: sheets}
	workbook.SortSheets([]string{"IP", "Personnel", "System", "Matrix"})
	return workbook
}

func attributeKeys(entries []models.LedgerEntry) []string {
	keysMap := make(map[string]struct{})
	for _, entry := range entries {
		for key := range entry.Attributes {
			keysMap[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(keysMap))
	for key := range keysMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *Server) buildCartesianRows() [][]string {
	ips := s.Store.ListEntries(models.LedgerTypeIP)
	personnel := s.Store.ListEntries(models.LedgerTypePersonnel)
	systems := s.Store.ListEntries(models.LedgerTypeSystem)

	rows := [][]string{}
	for _, system := range systems {
		linkedIPs := uniqueOrAll(system.Links[models.LedgerTypeIP], ips)
		linkedPersonnel := uniqueOrAll(system.Links[models.LedgerTypePersonnel], personnel)
		for _, ipID := range linkedIPs {
			ipName := lookupName(ipID, ips)
			for _, personID := range linkedPersonnel {
				personName := lookupName(personID, personnel)
				rows = append(rows, []string{ipName, personName, system.Name})
			}
		}
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"", "", ""})
	}
	return rows
}

func uniqueOrAll(ids []string, entries []models.LedgerEntry) []string {
	if len(ids) == 0 {
		out := make([]string, 0, len(entries))
		for _, entry := range entries {
			out = append(out, entry.ID)
		}
		return out
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		for _, entry := range entries {
			out = append(out, entry.ID)
		}
	}
	return out
}

func lookupName(id string, entries []models.LedgerEntry) string {
	for _, entry := range entries {
		if entry.ID == id {
			return entry.Name
		}
	}
	return id
}

func sheetNameForType(typ models.LedgerType) string {
	switch typ {
	case models.LedgerTypeIP:
		return "IP"
	case models.LedgerTypePersonnel:
		return "Personnel"
	case models.LedgerTypeSystem:
		return "System"
	default:
		return strings.Title(string(typ))
	}
}
