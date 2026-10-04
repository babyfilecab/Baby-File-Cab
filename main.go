package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/argon2"
)

const (
	appTitle             = "BabyFileCab — Local Document Management"
	permanentFolder      = "Permanent Folder"
	notesFile            = "notes.txt"
	richNotesFile        = ".notes-rich.html"
	auditFile            = ".audit.jsonl"
	appAuditFile         = ".babyfilecab-audit.jsonl"
	clientProfileFile    = ".client-profile.json"
	documentMetadataFile = ".document-metadata.json"
	firmCalendarFile     = ".firm-calendar-2026.json"
	engagementLogoFile   = ".engagement-letter-logo.jpg"
)

//go:embed all:frontend/dist
var embeddedAssets embed.FS

type App struct {
	ctx         context.Context
	storageRoot string
	dataRoot    string

	authMu         sync.RWMutex
	currentUser    *UserProfile
	currentCompany *CompanyProfile
	vaultKey       []byte
	vaultMu        sync.RWMutex
	lastActivity   time.Time
	autoLockMinutes int
	calendarMu     sync.Mutex
	auditMu        sync.Mutex
	tempMu         sync.Mutex
	tempDirs       map[string]struct{}
}

const (
	passwordIterations = 210000
	passwordKeyLength  = 32
	passwordSaltLength = 24
)

type CompanyProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DataDir   string `json:"dataDir"`
	CreatedAt string `json:"createdAt"`
}

type UserProfile struct {
	Username      string `json:"username"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Role          string `json:"role"`
	CompanyID     string `json:"companyId"`
	CompanyName   string `json:"companyName"`
	StreetAddress string `json:"streetAddress,omitempty"`
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	ZIP           string `json:"zip,omitempty"`
	CAF           string `json:"caf,omitempty"`
	PTIN          string `json:"ptin,omitempty"`
	Telephone     string `json:"telephone,omitempty"`
	Fax           string `json:"fax,omitempty"`
	CreatedAt     string `json:"createdAt"`
	VaultStatus   string `json:"vaultStatus,omitempty"`
}

type UserRegistration struct {
	CompanyName     string `json:"companyName"`
	Username        string `json:"username"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
	Role            string `json:"role"`
	StreetAddress   string `json:"streetAddress"`
	City            string `json:"city"`
	State           string `json:"state"`
	ZIP             string `json:"zip"`
	CAF             string `json:"caf"`
	PTIN            string `json:"ptin"`
	Telephone       string `json:"telephone"`
	Fax             string `json:"fax"`
}

// ExistingCompanyUserRegistration is used when an administrator authorizes
// another Staff or Administrator account to join an existing company.
// The administrator password is verified only for this operation and is never stored.
type ExistingCompanyUserRegistration struct {
	CompanyID       string `json:"companyId"`
	AdminUsername   string `json:"adminUsername"`
	AdminPassword   string `json:"adminPassword"`
	Username        string `json:"username"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
	Role            string `json:"role"`
	StreetAddress   string `json:"streetAddress"`
	City            string `json:"city"`
	State           string `json:"state"`
	ZIP             string `json:"zip"`
	CAF             string `json:"caf"`
	PTIN            string `json:"ptin"`
	Telephone       string `json:"telephone"`
	Fax             string `json:"fax"`
}

type storedUser struct {
	UserProfile
	PasswordSalt string `json:"passwordSalt"`
	PasswordHash string `json:"passwordHash"`
	Iterations   int    `json:"iterations"`
	PasswordKDF  string `json:"passwordKdf,omitempty"`
	KeyWrapID string `json:"keyWrapId,omitempty"`
}

type TreeNode struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	ClientID string     `json:"clientId,omitempty"`
	Type     string     `json:"type"`
	Path     string     `json:"path"`
	Children []TreeNode `json:"children,omitempty"`
}

type ClientProperties struct {
	Name       string `json:"name"`
	ClientID   string `json:"clientId"`
	Path       string `json:"path"`
	Address    string `json:"address,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
	TaxID      string `json:"taxId,omitempty"`
	ReturnType string `json:"returnType,omitempty"`
}

type ClientProfile struct {
	Address    string `json:"address"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	TaxID      string `json:"taxId"`
	ReturnType string `json:"returnType"`
}

type FirmCalendarAssignment struct {
	Date       string `json:"date"`
	ClientID   string `json:"clientId"`
	ClientName string `json:"clientName"`
	ClientPath string `json:"clientPath,omitempty"`
	ReturnType string `json:"returnType,omitempty"`
 Status string `json:"status,omitempty"`
 PreparerUsername string `json:"preparerUsername,omitempty"`
}

type storedFirmCalendarAssignment struct {
	Date       string `json:"date"`
	ClientID   string `json:"clientId"`
	ClientName string `json:"clientName"`
	ReturnType string `json:"returnType,omitempty"`
 Status string `json:"status,omitempty"`
 PreparerUsername string `json:"preparerUsername,omitempty"`
}

type ClientCommunication struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	ReturnType string `json:"returnType"`
	Path       string `json:"path"`
}

type EngagementLetterRequest struct {
	ClientPath       string `json:"clientPath"`
	PreparerUsername string `json:"preparerUsername"`
	Service          string `json:"service"`
	TaxYear          string `json:"taxYear"`
	FeeType          string `json:"feeType"`
	FeeValue         string `json:"feeValue"`
	LetterDate       string `json:"letterDate"`
	OutputPath       string `json:"outputPath"`
}

type EngagementLetterLogoInfo struct {
	Path    string `json:"path"`
	DataURL string `json:"dataURL"`
}

type engagementLetterData struct {
	CompanyName     string
	ClientID        string
	ClientName      string
	ClientPath      string
	ClientAddress   string
	ClientPhone     string
	ClientEmail     string
	PreparerName    string
	PreparerAddress []string
	PreparerPhone   string
	PreparerEmail   string
	Service         string
	ServiceDesc     string
	TaxYear         string
	FeeType         string
	FeeValue        string
	LetterDate      string
	LogoPath        string
	LogoWidth       int
	LogoHeight      int
}

type engagementLetterSection struct {
	Title string
	Body  string
}

type InvoiceRequest struct {
	ClientPath       string `json:"clientPath"`
	PreparerUsername string `json:"preparerUsername"`
	Service          string `json:"service"`
	TaxYear          string `json:"taxYear"`
	InvoiceNumber    string `json:"invoiceNumber"`
	InvoiceDate      string `json:"invoiceDate"`
	DueDate          string `json:"dueDate"`
	FeeType          string `json:"feeType"`
	FeeValue         string `json:"feeValue"`
	Hours            string `json:"hours"`
	Description      string `json:"description"`
	Notes            string `json:"notes"`
	OutputPath       string `json:"outputPath"`
}

type invoiceData struct {
	CompanyName     string
	ClientID        string
	ClientName      string
	ClientPath      string
	ClientAddress   string
	ClientPhone     string
	ClientEmail     string
	PreparerName    string
	PreparerAddress []string
	PreparerPhone   string
	PreparerEmail   string
	Service         string
	ServiceDesc     string
	TaxYear         string
	InvoiceNumber   string
	InvoiceDate     string
	DueDate         string
	FeeType         string
	Rate            float64
	Hours           float64
	Total           float64
	Description     string
	Notes           string
	LogoPath        string
	LogoWidth       int
	LogoHeight      int
}

type engagementPDFBlock struct {
	Text   string
	Bold   bool
	Size   float64
	Align  string
	Before float64
	After  float64
}

type DocumentProperties struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ClientName  string `json:"clientName"`
	ClientID    string `json:"clientId"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	Modified    string `json:"modified"`
}

type GlobalSearchResult struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ClientName  string `json:"clientName"`
	ClientID    string `json:"clientId"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Match       string `json:"match"`
}

type NotesDocument struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Details   string `json:"details"`
}

// AppAuditEntry is a company-wide audit record. Unlike the per-client audit
// file, it identifies the user and affected client so the Tools > Audit Trail
// window can show activity across the entire BabyFileCab company workspace.
type AppAuditEntry struct {
	Timestamp  string `json:"timestamp"`
	Username   string `json:"username,omitempty"`
	UserName   string `json:"userName,omitempty"`
	Role       string `json:"role,omitempty"`
	ClientID   string `json:"clientId,omitempty"`
	ClientName string `json:"clientName,omitempty"`
	Action     string `json:"action"`
	Details    string `json:"details,omitempty"`
}

type Preview struct {
	Kind      string     `json:"kind"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Text      string     `json:"text,omitempty"`
	ImageData string     `json:"imageData,omitempty"`
	Rows      [][]string `json:"rows,omitempty"`
	PageCount int        `json:"pageCount,omitempty"`
	Message   string     `json:"message,omitempty"`
}

type PDFSearchMatch struct {
	Page       int     `json:"page"`
	Snippet    string  `json:"snippet"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	PageWidth  float64 `json:"pageWidth"`
	PageHeight float64 `json:"pageHeight"`
}

type PDFSearchResponse struct {
	Query      string           `json:"query"`
	Total      int              `json:"total"`
	Results    []PDFSearchMatch `json:"results"`
	Searchable bool             `json:"searchable"`
	Message    string           `json:"message,omitempty"`
}

type pdfBBoxWord struct {
	Text string  `xml:",chardata"`
	XMin float64 `xml:"xMin,attr"`
	YMin float64 `xml:"yMin,attr"`
	XMax float64 `xml:"xMax,attr"`
	YMax float64 `xml:"yMax,attr"`
}

type pdfBBoxPage struct {
	Width  float64       `xml:"width,attr"`
	Height float64       `xml:"height,attr"`
	Words  []pdfBBoxWord `xml:"word"`
}

type pdfBBoxDocument struct {
	Pages []pdfBBoxPage `xml:"body>doc>page"`
}

func main() {
	assets, err := fs.Sub(embeddedAssets, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:            appTitle,
		Width:            1420,
		Height:           900,
		MinWidth:         1050,
		MinHeight:        680,
		BackgroundColour: &options.RGBA{R: 250, G: 252, B: 255, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
		},
		Bind: []interface{}{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func NewApp() (*App, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(home, "BabyFileCabData")
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(root, 0700); err != nil {
		return nil, err
	}
	app := &App{storageRoot: root, dataRoot: root, autoLockMinutes: 15}
 if data, err := os.ReadFile(filepath.Join(root, ".autolock-minutes")); err == nil {
  var minutes int
  if json.Unmarshal(data, &minutes) == nil && validAutoLockMinutes(minutes) {
   app.autoLockMinutes = minutes
  }
 }
	if err := os.RemoveAll(filepath.Join(root, ".plaintext-preview")); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, ".plaintext-preview"), 0700); err != nil {
		return nil, err
	}
	if err := app.migrateLegacyCompanyData(); err != nil {
		return nil, err
	}
	if app.accountsReady() {
		_ = os.Remove(app.usersPath())
		_ = os.Remove(app.companiesPath())
	}
	return app, nil
}

func validAutoLockMinutes(minutes int) bool {
 return minutes == 15 || minutes == 30 || minutes == 45
}

// Caller holds authMu for reading or writing.
func (a *App) idleTimeoutLocked() time.Duration {
 minutes := a.autoLockMinutes
 if !validAutoLockMinutes(minutes) { minutes = 15 }
 return time.Duration(minutes) * time.Minute
}

func (a *App) GetAutoLockMinutes() int {
 a.authMu.RLock()
 defer a.authMu.RUnlock()
 return int(a.idleTimeoutLocked() / time.Minute)
}

func (a *App) SetAutoLockMinutes(minutes int) error {
 a.authMu.Lock()
 defer a.authMu.Unlock()
 if !validAutoLockMinutes(minutes) { return errors.New("choose 15, 30, or 45 minutes") }
 if a.currentUser == nil || time.Since(a.lastActivity) >= a.idleTimeoutLocked() {
  return errors.New("sign in to BabyFileCab first")
 }
 data, err := json.Marshal(minutes)
 if err != nil { return err }
 if err := writeAtomicPrivate(filepath.Join(a.storageRoot, ".autolock-minutes"), data); err != nil { return err }
 a.autoLockMinutes = minutes
 a.lastActivity = time.Now()
 return nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startSessionMonitor(ctx)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		lastTickUnix := time.Now().UnixNano()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				nowUnix := time.Now().UnixNano()
				gap := time.Duration(nowUnix - lastTickUnix)
				lastTickUnix = nowUnix
				a.authMu.RLock()
				expired := a.currentUser != nil && (time.Since(a.lastActivity) >= a.idleTimeoutLocked() || gap >= a.idleTimeoutLocked())
				a.authMu.RUnlock()
				if expired {
					a.Logout()
				}
			}
		}
	}()
}

func (a *App) DataRoot() string {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	return a.dataRoot
}

func (a *App) HasUsers() bool {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	users, err := a.loadUsersUnlocked()
	return err == nil && len(users) > 0
}

func (a *App) CurrentUser() UserProfile {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	if a.currentUser == nil || time.Since(a.lastActivity) >= a.idleTimeoutLocked() {
		return UserProfile{}
	}
	return *a.currentUser
}

// ListCompanyUsers returns the user profiles that belong to the currently
// signed-in company. Password salts and hashes are never exposed to the UI.
func (a *App) ListCompanyUsers() ([]UserProfile, error) {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	if a.currentUser == nil || a.currentCompany == nil || time.Since(a.lastActivity) >= a.idleTimeoutLocked() {
		return nil, errors.New("sign in to BabyFileCab first")
	}
	users, err := a.loadUsersUnlocked()
	if err != nil {
		return nil, err
	}
	profiles := make([]UserProfile, 0)
	for _, u := range users {
		if u.CompanyID != a.currentCompany.ID {
			continue
		}
		profile := u.UserProfile
		if err := a.metadataJSONGet("user-profile", strings.ToLower(u.Username), &profile); err != nil {
			return nil, err
		}
		profile.CompanyName = a.currentCompany.Name
		keyPath, err := a.activeWrappedKeyPath(a.currentCompany.ID, u.Username)
		if err != nil { return nil, err }
		if _, err := os.Stat(keyPath); err == nil {
			profile.VaultStatus = "Enrolled"
		} else {
			profile.VaultStatus = "Pending"
		}
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(profiles[i].LastName + " " + profiles[i].FirstName + " " + profiles[i].Username))
		right := strings.ToLower(strings.TrimSpace(profiles[j].LastName + " " + profiles[j].FirstName + " " + profiles[j].Username))
		return left < right
	})
	return profiles, nil
}

// ListCompanies returns the locally registered companies so the sign-in screen
// can offer "Create User Under Existing Company". Client/document data is not
// exposed by this method.
func (a *App) ListCompanies() ([]CompanyProfile, error) {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	companies, err := a.loadCompaniesUnlocked()
	if err != nil {
		return nil, err
	}
	sort.Slice(companies, func(i, j int) bool {
		return strings.ToLower(companies[i].Name) < strings.ToLower(companies[j].Name)
	})
	return companies, nil
}

// RegisterUser creates a new local company and its first account. A company
// must start with an Administrator so additional users can later be authorized.
func (a *App) RegisterUser(req UserRegistration) (UserProfile, error) {
	normalizeUserRegistration(&req)
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	if req.CompanyName == "" {
		return UserProfile{}, errors.New("company name is required")
	}
	if err := validateUserRegistration(req); err != nil {
		return UserProfile{}, err
	}
	if req.Role != "administrator" {
		return UserProfile{}, errors.New("the first user for a new company must be an Administrator")
	}

	a.authMu.Lock()
	defer a.authMu.Unlock()

	users, err := a.loadUsersUnlocked()
	if err != nil {
		return UserProfile{}, err
	}
	if usernameExists(users, req.Username) {
		return UserProfile{}, errors.New("that username already exists")
	}
	companies, err := a.loadCompaniesUnlocked()
	if err != nil {
		return UserProfile{}, err
	}
	for _, company := range companies {
		if strings.EqualFold(strings.TrimSpace(company.Name), req.CompanyName) {
			return UserProfile{}, errors.New("a company with that name already exists; use Create User Under Existing Company instead")
		}
	}

	companyID, err := newCompanyID()
	if err != nil {
		return UserProfile{}, err
	}
	company := CompanyProfile{
		ID:        companyID,
		Name:      req.CompanyName,
		DataDir:   filepath.Join("companies", companyID),
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	companyRoot, err := a.companyDataRootUnlocked(company)
	if err != nil {
		return UserProfile{}, err
	}
	if err := os.MkdirAll(companyRoot, 0700); err != nil {
		return UserProfile{}, fmt.Errorf("create company data folder: %w", err)
	}

	entry, profile, err := buildStoredUser(req, company)
	if err != nil {
		return UserProfile{}, err
	}
	vaultKey, err := randomBytes(vaultKeySize)
	if err != nil {
		return UserProfile{}, err
	}
	if err := a.writeWrappedKey(company.ID, profile.Username, req.Password, vaultKey); err != nil {
		return UserProfile{}, err
	}
	if err := migrateCompanyData(companyRoot, a.storageRoot, company.ID, vaultKey); err != nil {
		return UserProfile{}, err
	}
	if err := migrateAuditToSQLite(companyRoot, a.storageRoot, vaultKey); err != nil {
		return UserProfile{}, err
	}
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		return UserProfile{}, err
	}
	profileDB, err := openVaultDB(companyRoot)
	if err != nil {
		return UserProfile{}, err
	}
	err = metadataPut(profileDB, vaultKey, "user-profile", strings.ToLower(profile.Username), profileBytes)
	_ = profileDB.Close()
	if err != nil {
		return UserProfile{}, err
	}
	companies = append(companies, company)
	if err := a.saveCompaniesUnlocked(companies); err != nil {
		return UserProfile{}, err
	}
	users = append(users, redactedUser(entry))
	if err := a.saveUsersUnlocked(users); err != nil {
		return UserProfile{}, err
	}
	if err := a.maybeMigrateAuthIndex(users); err != nil {
		return UserProfile{}, err
	}

	a.currentUser = &profile
	a.currentCompany = &company
	a.vaultMu.Lock()
	a.vaultKey = vaultKey
	a.vaultMu.Unlock()
	a.lastActivity = time.Now()
	a.dataRoot = companyRoot
	_ = a.appendAppAuditFor(companyRoot, profile, "", "", "Company created", fmt.Sprintf("%s created with first Administrator account @%s", company.Name, profile.Username))
	return profile, nil
}

// CreateUserUnderExistingCompany adds a user to an existing company only after
// an active Administrator for that same company supplies valid credentials.
// Every user in that company resolves to the same company dataRoot, so they see
// the same client tree and documents on this local BabyFileCab installation.
func (a *App) CreateUserUnderExistingCompany(req ExistingCompanyUserRegistration) (UserProfile, error) {
	companyID := strings.TrimSpace(req.CompanyID)
	adminUsername := strings.TrimSpace(req.AdminUsername)
	adminPassword := req.AdminPassword
	if companyID == "" {
		return UserProfile{}, errors.New("select a company")
	}
	if adminUsername == "" || adminPassword == "" {
		return UserProfile{}, errors.New("administrator username and password are required")
	}

	userReq := UserRegistration{
		Username: req.Username, FirstName: req.FirstName, LastName: req.LastName,
		Email: req.Email, Phone: req.Phone, Password: req.Password,
		ConfirmPassword: req.ConfirmPassword, Role: req.Role,
		StreetAddress: req.StreetAddress, City: req.City, State: req.State, ZIP: req.ZIP,
		CAF: req.CAF, PTIN: req.PTIN, Telephone: req.Telephone, Fax: req.Fax,
	}
	normalizeUserRegistration(&userReq)
	if err := validateUserRegistration(userReq); err != nil {
		return UserProfile{}, err
	}

	a.authMu.Lock()
	defer a.authMu.Unlock()

	companies, err := a.loadCompaniesUnlocked()
	if err != nil {
		return UserProfile{}, err
	}
	company, ok := findCompany(companies, companyID)
	if !ok {
		return UserProfile{}, errors.New("selected company no longer exists")
	}
	users, err := a.loadUsersUnlocked()
	if err != nil {
		return UserProfile{}, err
	}
	if usernameExists(users, userReq.Username) {
		return UserProfile{}, errors.New("that username already exists")
	}

	adminOK := false
	var adminProfile UserProfile
	for _, candidate := range users {
		if candidate.CompanyID != company.ID || !strings.EqualFold(candidate.Username, adminUsername) {
			continue
		}
		if strings.ToLower(candidate.Role) != "administrator" {
			return UserProfile{}, errors.New("that account is not an Administrator for the selected company")
		}
		ok, err := verifyStoredPassword(candidate, adminPassword)
		if err != nil {
			return UserProfile{}, err
		}
		adminOK = ok
		if ok {
			adminProfile = candidate.UserProfile
			adminProfile.CompanyName = company.Name
		}
		break
	}
	if !adminOK {
		return UserProfile{}, errors.New("administrator authorization failed")
	}

	entry, profile, err := buildStoredUser(userReq, company)
	if err != nil {
		return UserProfile{}, err
	}
	adminKey, err := a.unwrapKey(company.ID, adminUsername, adminPassword)
	if err != nil {
		return UserProfile{}, fmt.Errorf("administrator must unlock the company vault before adding users: %w", err)
	}
	if err := a.writeWrappedKey(company.ID, profile.Username, userReq.Password, adminKey); err != nil {
		return UserProfile{}, err
	}
	defer func() {
		for i := range adminKey {
			adminKey[i] = 0
		}
	}()
	root, err := a.companyDataRootUnlocked(company)
	if err != nil {
		return UserProfile{}, err
	}
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		return UserProfile{}, err
	}
	profileDB, err := openVaultDB(root)
	if err != nil {
		return UserProfile{}, err
	}
	err = metadataPut(profileDB, adminKey, "user-profile", strings.ToLower(profile.Username), profileBytes)
	_ = profileDB.Close()
	if err != nil {
		return UserProfile{}, err
	}
	users = append(users, redactedUser(entry))
	if err := a.saveUsersUnlocked(users); err != nil {
		return UserProfile{}, err
	}
	companyRoot, rootErr := a.companyDataRootUnlocked(company)
	if rootErr == nil {
		_ = a.appendAppAuditFor(companyRoot, adminProfile, "", "", "User account created", fmt.Sprintf("@%s created as %s", profile.Username, displayAuditRole(profile.Role)))
	}
	return profile, nil
}

func (a *App) Login(username, password string) (UserProfile, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return UserProfile{}, errors.New("enter your username and password")
	}

	a.authMu.Lock()
	defer a.authMu.Unlock()
	users, err := a.loadUsersUnlocked()
	if err != nil {
		return UserProfile{}, err
	}
	companies, err := a.loadCompaniesUnlocked()
	if err != nil {
		return UserProfile{}, err
	}
	for index, u := range users {
		if !strings.EqualFold(u.Username, username) {
			continue
		}
		ok, err := verifyStoredPassword(u, password)
		if err != nil {
			return UserProfile{}, err
		}
		if !ok {
			break
		}
		company, found := findCompany(companies, u.CompanyID)
		if !found {
			return UserProfile{}, errors.New("the company assigned to this account could not be found")
		}
		vaultKey, keyErr := a.unwrapKey(company.ID, u.Username, password)
		if keyErr != nil {
            if errors.Is(keyErr, os.ErrNotExist) {
                return UserProfile{}, errors.New("company vault key is missing: restore the encrypted key file from a complete backup, or ask an administrator who can already unlock this company to enroll your account; no replacement key was created")
            }
            return UserProfile{}, keyErr
        }

		if u.PasswordKDF == "" {
			// Upgrade only after a successful legacy password check. An interrupted
			// write leaves the old verifier usable on the next login.
			salt := make([]byte, passwordSaltLength)
			if _, err := rand.Read(salt); err != nil {
				return UserProfile{}, err
			}
			hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, passwordKeyLength)
			users[index].PasswordSalt = base64.StdEncoding.EncodeToString(salt)
			users[index].PasswordHash = base64.StdEncoding.EncodeToString(hash)
			users[index].PasswordKDF = "argon2id-v1"
			users[index].Iterations = 0
			if err := a.saveUsersUnlocked(users); err != nil {
				return UserProfile{}, err
			}
		}
		root, err := a.companyDataRootUnlocked(company)
		if err != nil {
			return UserProfile{}, err
		}
		if err := os.MkdirAll(root, 0700); err != nil {
			return UserProfile{}, fmt.Errorf("open company data folder: %w", err)
		}
		if err := migrateCompanyData(root, a.storageRoot, company.ID, vaultKey); err != nil {
			zeroBytes(vaultKey)
			return UserProfile{}, fmt.Errorf("migrate company vault: %w", err)
		}
		if err := migrateAuditToSQLite(root, a.storageRoot, vaultKey); err != nil {
			zeroBytes(vaultKey)
			return UserProfile{}, err
		}
		users, err = a.migrateCompanyUserProfiles(users, company, root, vaultKey)
		if err != nil {
			zeroBytes(vaultKey)
			return UserProfile{}, err
		}
		if err := a.maybeMigrateAuthIndex(users); err != nil {
			zeroBytes(vaultKey)
			return UserProfile{}, err
		}
		profileDB, err := openVaultDB(root)
		if err != nil {
			return UserProfile{}, err
		}
		profileBytes, err := metadataGet(profileDB, vaultKey, "user-profile", strings.ToLower(u.Username))
		_ = profileDB.Close()
		if err != nil {
			return UserProfile{}, err
		}
		var profile UserProfile
		if err := json.Unmarshal(profileBytes, &profile); err != nil {
			return UserProfile{}, err
		}
		profile.CompanyName = company.Name
		a.currentUser = &profile
		a.currentCompany = &company
		a.vaultMu.Lock()
		a.vaultKey = vaultKey
		a.vaultMu.Unlock()
		a.lastActivity = time.Now()
		a.dataRoot = root
		_ = a.appendAppAuditFor(root, profile, "", "", "Signed in", "BabyFileCab session started")
		return profile, nil
	}
	return UserProfile{}, errors.New("invalid username or password")
}

func (a *App) Logout() {
	a.authMu.Lock()
	var user UserProfile
	root := a.dataRoot
	if a.currentUser != nil {
		user = *a.currentUser
	}
	if user.Username != "" && root != "" {
		_ = a.appendAppAuditFor(root, user, "", "", "Signed out", "BabyFileCab session ended")
	}
	a.currentUser = nil
	a.currentCompany = nil
	a.vaultMu.Lock()
	for i := range a.vaultKey {
		a.vaultKey[i] = 0
	}
	a.vaultKey = nil
	a.vaultMu.Unlock()
	a.lastActivity = time.Time{}
	a.dataRoot = a.storageRoot
	a.authMu.Unlock()
	a.removeDecryptedTemps()
}

// EnrollExistingUser adds a separately wrapped company key only after the
// administrator has unlocked the vault and the target user proves their own
// existing password. Neither person has to reveal a password to the other.
func (a *App) EnrollExistingUser(username, password string) error {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	if a.currentUser == nil || a.currentCompany == nil || a.currentUser.Role != "administrator" || len(a.vaultKey) != vaultKeySize {
		return errors.New("an administrator must unlock the company vault first")
	}
	if time.Since(a.lastActivity) >= a.idleTimeoutLocked() {
		return errors.New("the company vault is locked")
	}
	users, err := a.loadUsersUnlocked()
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.CompanyID != a.currentCompany.ID || !strings.EqualFold(user.Username, strings.TrimSpace(username)) {
			continue
		}
		if user.KeyWrapID != "" { return errors.New("user is already enrolled; restore the encrypted vault-key file if it is missing") }
		if _, err := os.Stat(a.wrappedKeyPath(user.CompanyID, user.Username)); err == nil {
			return errors.New("user is already enrolled")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		ok, err := verifyStoredPassword(user, password)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("incorrect user password")
		}
		if err := a.writeWrappedKey(user.CompanyID, user.Username, password, a.vaultKey); err != nil {
			return err
		}
		key, err := a.unwrapKey(user.CompanyID, user.Username, password)
		if err != nil {
			_ = os.Remove(a.wrappedKeyPath(user.CompanyID, user.Username))
			return err
		}
		verified := subtle.ConstantTimeCompare(key, a.vaultKey) == 1
		for i := range key {
			key[i] = 0
		}
		if !verified {
			_ = os.Remove(a.wrappedKeyPath(user.CompanyID, user.Username))
			return errors.New("vault enrollment verification failed")
		}
		_ = a.appendAppAuditFor(a.dataRoot, *a.currentUser, "", "", "Vault user enrolled", "@"+user.Username)
		return nil
	}
	return errors.New("user does not belong to this company")
}

func (a *App) usersPath() string {
	return filepath.Join(a.storageRoot, ".users.json")
}

func (a *App) companiesPath() string {
	return filepath.Join(a.storageRoot, ".companies.json")
}

func (a *App) loadUsersUnlocked() ([]storedUser, error) {
	if a.accountsReady() {
		return a.loadUsersSQLite()
	}
	data, err := os.ReadFile(a.usersPath())
	if errors.Is(err, os.ErrNotExist) {
		return []storedUser{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read users: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []storedUser{}, nil
	}
	var users []storedUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("read users: %w", err)
	}
	return users, nil
}

func (a *App) saveUsersUnlocked(users []storedUser) error {
	if a.accountsReady() {
		return a.saveUsersSQLite(users)
	}
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("save users: %w", err)
	}
	if err := writeAtomicPrivate(a.usersPath(), data); err != nil { return err }
	dir, err := os.Open(a.storageRoot)
	if err != nil { return err }
	defer dir.Close()
	return dir.Sync()
}

func (a *App) loadCompaniesUnlocked() ([]CompanyProfile, error) {
	if a.accountsReady() {
		return a.loadCompaniesSQLite()
	}
	data, err := os.ReadFile(a.companiesPath())
	if errors.Is(err, os.ErrNotExist) {
		return []CompanyProfile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read companies: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []CompanyProfile{}, nil
	}
	var companies []CompanyProfile
	if err := json.Unmarshal(data, &companies); err != nil {
		return nil, fmt.Errorf("read companies: %w", err)
	}
	return companies, nil
}

func (a *App) saveCompaniesUnlocked(companies []CompanyProfile) error {
	if a.accountsReady() {
		return a.saveCompaniesSQLite(companies)
	}
	data, err := json.MarshalIndent(companies, "", "  ")
	if err != nil {
		return fmt.Errorf("save companies: %w", err)
	}
	path := a.companiesPath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("save companies: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("save companies: %w", err)
	}
	return nil
}

func normalizeUserRegistration(req *UserRegistration) {
	req.Username = strings.TrimSpace(req.Username)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	req.StreetAddress = strings.TrimSpace(req.StreetAddress)
	req.City = strings.TrimSpace(req.City)
	req.State = strings.TrimSpace(req.State)
	req.ZIP = strings.TrimSpace(req.ZIP)
	req.CAF = strings.TrimSpace(req.CAF)
	req.PTIN = strings.TrimSpace(req.PTIN)
	req.Telephone = strings.TrimSpace(req.Telephone)
	req.Fax = strings.TrimSpace(req.Fax)
}

func buildStoredUser(req UserRegistration, company CompanyProfile) (storedUser, UserProfile, error) {
	salt := make([]byte, passwordSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return storedUser{}, UserProfile{}, fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(req.Password), salt, 3, 64*1024, 4, passwordKeyLength)
	profile := UserProfile{
		Username: req.Username, FirstName: req.FirstName, LastName: req.LastName,
		Email: req.Email, Phone: req.Phone, Role: req.Role,
		CompanyID: company.ID, CompanyName: company.Name,
		StreetAddress: req.StreetAddress, City: req.City, State: req.State, ZIP: req.ZIP,
		CAF: req.CAF, PTIN: req.PTIN, Telephone: req.Telephone, Fax: req.Fax,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	entry := storedUser{
		UserProfile: profile, PasswordSalt: base64.StdEncoding.EncodeToString(salt),
		PasswordHash: base64.StdEncoding.EncodeToString(hash), PasswordKDF: "argon2id-v1",
	}
	return entry, profile, nil
}

func verifyStoredPassword(user storedUser, password string) (bool, error) {
	salt, err := base64.StdEncoding.DecodeString(user.PasswordSalt)
	if err != nil {
		return false, errors.New("stored account is damaged")
	}
	want, err := base64.StdEncoding.DecodeString(user.PasswordHash)
	if err != nil || len(want) == 0 {
		return false, errors.New("stored account is damaged")
	}
	if user.PasswordKDF == "argon2id-v1" {
		if len(salt) != passwordSaltLength || len(want) != passwordKeyLength {
			return false, errors.New("stored account is damaged")
		}
		got := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, passwordKeyLength)
		return subtle.ConstantTimeCompare(got, want) == 1, nil
	}
	if user.PasswordKDF != "" {
		return false, errors.New("unsupported password format")
	}
	iterations := user.Iterations
	if iterations <= 0 {
		iterations = passwordIterations
	}
	got := pbkdf2SHA256([]byte(password), salt, iterations, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func usernameExists(users []storedUser, username string) bool {
	for _, user := range users {
		if strings.EqualFold(user.Username, username) {
			return true
		}
	}
	return false
}

func findCompany(companies []CompanyProfile, id string) (CompanyProfile, bool) {
	for _, company := range companies {
		if company.ID == id {
			return company, true
		}
	}
	return CompanyProfile{}, false
}

func newCompanyID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate company ID: %w", err)
	}
	return fmt.Sprintf("company-%x", b), nil
}

func (a *App) companyDataRootUnlocked(company CompanyProfile) (string, error) {
	if strings.TrimSpace(company.DataDir) == "" || company.DataDir == "." {
		return a.storageRoot, nil
	}
	root, err := filepath.Abs(filepath.Join(a.storageRoot, company.DataDir))
	if err != nil {
		return "", err
	}
	storage, _ := filepath.Abs(a.storageRoot)
	if root != storage && !strings.HasPrefix(root, storage+string(os.PathSeparator)) {
		return "", errors.New("company data folder is outside BabyFileCab storage")
	}
	return root, nil
}

// migrateLegacyCompanyData keeps accounts and clients created by the previous
// sign-in build working. Legacy users are assigned to one shared company whose
// data directory remains the existing ~/BabyFileCabData root, so no client
// folders are moved or renamed during this upgrade.
func (a *App) migrateLegacyCompanyData() error {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	users, err := a.loadUsersUnlocked()
	if err != nil {
		return err
	}
	companies, err := a.loadCompaniesUnlocked()
	if err != nil {
		return err
	}
	needsMigration := false
	for _, user := range users {
		if strings.TrimSpace(user.CompanyID) == "" {
			needsMigration = true
			break
		}
	}
	if !needsMigration {
		return nil
	}

	const legacyID = "company-existing-local"
	legacy, found := findCompany(companies, legacyID)
	if !found {
		legacy = CompanyProfile{
			ID: legacyID, Name: "Existing BabyFileCab Company", DataDir: ".",
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		companies = append(companies, legacy)
	}
	for i := range users {
		if strings.TrimSpace(users[i].CompanyID) == "" {
			users[i].CompanyID = legacy.ID
			users[i].CompanyName = legacy.Name
		}
	}
	if err := a.saveCompaniesUnlocked(companies); err != nil {
		return err
	}
	return a.saveUsersUnlocked(users)
}

func validateUserRegistration(req UserRegistration) error {
	if len(req.Username) < 3 || len(req.Username) > 40 {
		return errors.New("username must be between 3 and 40 characters")
	}
	for _, r := range req.Username {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			continue
		}
		return errors.New("username may contain only letters, numbers, periods, underscores, and hyphens")
	}
	if req.FirstName == "" || req.LastName == "" {
		return errors.New("first name and last name are required")
	}
	if req.Email == "" {
		return errors.New("email address is required")
	}
	if parsed, err := mail.ParseAddress(req.Email); err != nil || !strings.EqualFold(strings.TrimSpace(parsed.Address), req.Email) {
		return errors.New("enter a valid email address")
	}
	if req.Phone == "" {
		return errors.New("phone number is required")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if req.Password != req.ConfirmPassword {
		return errors.New("password and confirm password do not match")
	}
	if req.Role != "staff" && req.Role != "administrator" {
		return errors.New("role must be Staff or Administrator")
	}
	return nil
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	if iterations < 1 {
		iterations = 1
	}
	hashLength := sha256.Size
	blocks := (keyLength + hashLength - 1) / hashLength
	result := make([]byte, 0, blocks*hashLength)
	var counter [4]byte
	for block := 1; block <= blocks; block++ {
		binary.BigEndian.PutUint32(counter[:], uint32(block))
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write(counter[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		result = append(result, t...)
	}
	return result[:keyLength]
}

func (a *App) GetClientProperties(path string) (ClientProperties, error) {
	path, err := a.requireType(path, "client")
	if err != nil {
		return ClientProperties{}, err
	}
	clientID, displayName, ok := parseClientDirName(filepath.Base(path))
	if !ok {
		return ClientProperties{}, errors.New("invalid client folder")
	}
	profile, _ := a.readClientProfile(path)
	return ClientProperties{
		Name:       displayName,
		ClientID:   clientID,
		Path:       path,
		Address:    profile.Address,
		Phone:      profile.Phone,
		Email:      profile.Email,
		TaxID:      profile.TaxID,
		ReturnType: profile.ReturnType,
	}, nil
}

func (a *App) UpdateClientProfile(path, address, phone, email, taxID, returnType string) error {
	path, err := a.requireType(path, "client")
	if err != nil {
		return err
	}
	returnType = strings.TrimSpace(returnType)
	switch returnType {
	case "1040", "1065", "1120S", "1120":
	default:
		return errors.New("return type must be 1040, 1065, 1120S, or 1120")
	}
	profile := ClientProfile{
		Address:    strings.TrimSpace(address),
		Phone:      strings.TrimSpace(phone),
		Email:      strings.TrimSpace(email),
		TaxID:      strings.TrimSpace(taxID),
		ReturnType: returnType,
	}
	if err := a.writeClientProfile(path, profile); err != nil {
		return err
	}
	return a.appendAudit(path, "Client profile updated", "Address, phone, email, tax ID, or return type was updated")
}

func (a *App) GetClientCommunications() ([]ClientCommunication, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(a.dataRoot)
	if err != nil {
		return nil, err
	}
	out := make([]ClientCommunication, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !isClientDirName(entry.Name()) {
			continue
		}
		clientPath := filepath.Join(a.dataRoot, entry.Name())
		_, displayName, ok := parseClientDirName(entry.Name())
		if !ok {
			continue
		}
		profile, _ := a.readClientProfile(clientPath)
		out = append(out, ClientCommunication{
			Name:       displayName,
			Email:      profile.Email,
			Phone:      profile.Phone,
			ReturnType: profile.ReturnType,
			Path:       clientPath,
		})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// GetFirmCalendar2026 returns the company-shared client schedule for calendar year 2026.
// The calendar file lives inside the signed-in company's local data directory, so every
// local BabyFileCab user in the same company sees the same assignments.
func (a *App) GetFirmCalendar2026() ([]FirmCalendarAssignment, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	a.calendarMu.Lock()
	defer a.calendarMu.Unlock()
	stored, err := a.loadFirmCalendarUnlocked()
	if err != nil {
		return nil, err
	}
	return a.resolveFirmCalendarAssignmentsUnlocked(stored)
}

func (a *App) AssignClientToFirmCalendar2026(date, clientPath string) ([]FirmCalendarAssignment, error) {
 return a.AssignClientToFirmCalendarForPreparer2026(date, clientPath, "")
}

func (a *App) AssignClientToFirmCalendarForPreparer2026(date, clientPath, preparerUsername string) ([]FirmCalendarAssignment, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	if err := validateFirmCalendarDate(date); err != nil {
		return nil, err
	}
	clientPath, err := a.requireType(clientPath, "client")
	if err != nil {
		return nil, err
	}
	props, err := a.GetClientProperties(clientPath)
	if err != nil {
		return nil, err
	}

 preparerUsername, err = a.validateCalendarPreparer(preparerUsername)
 if err != nil { return nil, err }

	a.calendarMu.Lock()
	defer a.calendarMu.Unlock()
	stored, err := a.loadFirmCalendarUnlocked()
	if err != nil {
		return nil, err
	}
	for _, item := range stored {
		if item.Date == date && item.ClientID == props.ClientID {
			return a.resolveFirmCalendarAssignmentsUnlocked(stored)
		}
	}
	stored = append(stored, storedFirmCalendarAssignment{
		Date:       date,
		ClientID:   props.ClientID,
		ClientName: props.Name,
		ReturnType: props.ReturnType,
        PreparerUsername: preparerUsername,
	})
	if err := a.saveFirmCalendarUnlocked(stored); err != nil {
		return nil, err
	}
	_ = a.appendAudit(clientPath, "Firm Calendar assignment added", date)
	return a.resolveFirmCalendarAssignmentsUnlocked(stored)
}

func (a *App) RemoveClientFromFirmCalendar2026(date, clientID string) ([]FirmCalendarAssignment, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	if err := validateFirmCalendarDate(date); err != nil {
		return nil, err
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, errors.New("client ID is required")
	}

	a.calendarMu.Lock()
	defer a.calendarMu.Unlock()
	stored, err := a.loadFirmCalendarUnlocked()
	if err != nil {
		return nil, err
	}
	filtered := make([]storedFirmCalendarAssignment, 0, len(stored))
	var removed *storedFirmCalendarAssignment
	for _, item := range stored {
		if item.Date == date && item.ClientID == clientID {
			copyItem := item
			removed = &copyItem
			continue
		}
		filtered = append(filtered, item)
	}
	if err := a.saveFirmCalendarUnlocked(filtered); err != nil {
		return nil, err
	}
	if removed != nil {
		if clientPath := a.clientPathByID(clientID); clientPath != "" {
			_ = a.appendAudit(clientPath, "Firm Calendar assignment removed", date)
		} else {
			_ = a.appendAppAuditCurrent(clientID, removed.ClientName, "Firm Calendar assignment removed", date)
		}
	}
	return a.resolveFirmCalendarAssignmentsUnlocked(filtered)
}

// ReassignClientInFirmCalendar2026 moves one scheduled client from one 2026 date
// to another. If the client is already assigned to the destination date, the
// original assignment is simply removed so the schedule never contains duplicates.
func (a *App) ReassignClientInFirmCalendar2026(oldDate, newDate, clientID string) ([]FirmCalendarAssignment, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	if err := validateFirmCalendarDate(oldDate); err != nil {
		return nil, err
	}
	if err := validateFirmCalendarDate(newDate); err != nil {
		return nil, err
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, errors.New("client ID is required")
	}
	if oldDate == newDate {
		return a.GetFirmCalendar2026()
	}

	a.calendarMu.Lock()
	defer a.calendarMu.Unlock()
	stored, err := a.loadFirmCalendarUnlocked()
	if err != nil {
		return nil, err
	}

	var moved *storedFirmCalendarAssignment
	filtered := make([]storedFirmCalendarAssignment, 0, len(stored))
	destinationAlreadyExists := false
	for _, item := range stored {
		if item.Date == oldDate && item.ClientID == clientID {
			copyItem := item
			moved = &copyItem
			continue
		}
		if item.Date == newDate && item.ClientID == clientID {
			destinationAlreadyExists = true
		}
		filtered = append(filtered, item)
	}
	if moved == nil {
		return nil, errors.New("calendar assignment was not found")
	}
	if !destinationAlreadyExists {
		moved.Date = newDate
		filtered = append(filtered, *moved)
	}
	if err := a.saveFirmCalendarUnlocked(filtered); err != nil {
		return nil, err
	}
	if clientPath := a.clientPathByID(clientID); clientPath != "" {
		_ = a.appendAudit(clientPath, "Firm Calendar assignment moved", fmt.Sprintf("%s → %s", oldDate, newDate))
	} else {
		_ = a.appendAppAuditCurrent(clientID, moved.ClientName, "Firm Calendar assignment moved", fmt.Sprintf("%s → %s", oldDate, newDate))
	}
	return a.resolveFirmCalendarAssignmentsUnlocked(filtered)
}

func validateFirmCalendarDate(value string) error {
	value = strings.TrimSpace(value)
	d, err := time.Parse("2006-01-02", value)
	if err != nil {
		return errors.New("calendar date must be in YYYY-MM-DD format")
	}
	if d.Year() != 2026 {
		return errors.New("Firm Calendar assignments are currently limited to calendar year 2026")
	}
	return nil
}

func (a *App) firmCalendarPath() string {
	return filepath.Join(a.dataRoot, firmCalendarFile)
}

func (a *App) loadFirmCalendarUnlocked() ([]storedFirmCalendarAssignment, error) {
	var items []storedFirmCalendarAssignment
	err := a.metadataJSONGet("firm-calendar", "all", &items)
	if errors.Is(err, os.ErrNotExist) {
		return []storedFirmCalendarAssignment{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read firm calendar: %w", err)
	}
	return items, nil
}

func (a *App) saveFirmCalendarUnlocked(items []storedFirmCalendarAssignment) error {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Date == items[j].Date {
			return strings.ToLower(items[i].ClientName) < strings.ToLower(items[j].ClientName)
		}
		return items[i].Date < items[j].Date
	})
	return a.metadataJSONPut("firm-calendar", "all", items)
}

func (a *App) resolveFirmCalendarAssignmentsUnlocked(stored []storedFirmCalendarAssignment) ([]FirmCalendarAssignment, error) {
	active := make(map[string]ClientProperties)
	entries, err := os.ReadDir(a.dataRoot)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !isClientDirName(entry.Name()) {
			continue
		}
		clientID, displayName, ok := parseClientDirName(entry.Name())
		if !ok {
			continue
		}
		path := filepath.Join(a.dataRoot, entry.Name())
		profile, _ := a.readClientProfile(path)
		active[clientID] = ClientProperties{Name: displayName, ClientID: clientID, Path: path, ReturnType: profile.ReturnType}
	}

	out := make([]FirmCalendarAssignment, 0, len(stored))
	for _, item := range stored {
		resolved := FirmCalendarAssignment{Date: item.Date, ClientID: item.ClientID, ClientName: item.ClientName, ReturnType: item.ReturnType, Status: item.Status, PreparerUsername: item.PreparerUsername}
		if props, ok := active[item.ClientID]; ok {
			resolved.ClientName = props.Name
			resolved.ClientPath = props.Path
			resolved.ReturnType = props.ReturnType
		}
		if strings.TrimSpace(resolved.ClientName) == "" {
			resolved.ClientName = "Client " + item.ClientID
		}
		out = append(out, resolved)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date == out[j].Date {
			return strings.ToLower(out[i].ClientName) < strings.ToLower(out[j].ClientName)
		}
		return out[i].Date < out[j].Date
	})
	return out, nil
}

func (a *App) BackupClient(path string) (string, error) {
	path, err := a.requireType(path, "client")
	if err != nil {
		return "", err
	}
	props, err := a.GetClientProperties(path)
	if err != nil {
		return "", err
	}
	base := sanitizeName(props.Name)
	if base == "" {
		base = "Client"
	}
	defaultName := fmt.Sprintf("BabyFileCab_%s_%s_%s.bfcbackup", props.ClientID, base, time.Now().Format("20060102-150405"))
	dest, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Back Up Client",
		DefaultFilename: defaultName,
		Filters:         []wailsruntime.FileFilter{{DisplayName: "Encrypted BabyFileCab Backup (*.bfcbackup)", Pattern: "*.bfcbackup"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(dest) == "" {
		return "", nil
	}
	if !strings.EqualFold(filepath.Ext(dest), ".bfcbackup") {
		dest += ".bfcbackup"
	}
	if err := a.appendAudit(path, "Backup created", filepath.Base(dest)); err != nil {
		return "", err
	}
	if err := a.encryptedClientBackup(path, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func (a *App) RestoreClient(path string) (string, error) {
	path, err := a.requireType(path, "client")
	if err != nil {
		return "", err
	}
	backupPath, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Restore Client from Backup",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Encrypted BabyFileCab Backup (*.bfcbackup)", Pattern: "*.bfcbackup"},
			{DisplayName: "Legacy BabyFileCab ZIP (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(backupPath) == "" {
		return "", nil
	}
	var restoreErr error
	if strings.EqualFold(filepath.Ext(backupPath), ".zip") {
		restoreErr = a.restoreLegacyClientZIP(path, backupPath)
	} else {
		restoreErr = a.restoreEncryptedClientBackup(path, backupPath)
	}
	if restoreErr != nil {
		return "", restoreErr
	}
	if err := a.appendAudit(path, "Client restored", filepath.Base(backupPath)); err != nil {
		return "", err
	}
	return backupPath, nil
}

func (a *App) ArchiveClient(path string) (string, error) {
	path, err := a.requireType(path, "client")
	if err != nil {
		return "", err
	}
	if err := a.appendAudit(path, "Client archived", "Client moved to the local BabyFileCab archive"); err != nil {
		return "", err
	}
	archiveRoot := filepath.Join(a.dataRoot, ".Archive")
	if err := os.MkdirAll(archiveRoot, 0700); err != nil {
		return "", err
	}
	dest := uniqueDestination(filepath.Join(archiveRoot, filepath.Base(path)))
	if err := os.Rename(path, dest); err != nil {
		return "", err
	}
	a.remapDocumentMetadataPrefix(path, dest)
	_ = a.metadataRenamePrefix("client-profile", a.documentMetadataKey(path), a.documentMetadataKey(dest))
	return dest, nil
}

func (a *App) ListArchivedClients() ([]TreeNode, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	archiveRoot := filepath.Join(a.dataRoot, ".Archive")
	entries, err := os.ReadDir(archiveRoot)
	if errors.Is(err, os.ErrNotExist) {
		return []TreeNode{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]TreeNode, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		clientID, displayName, ok := parseClientDirName(entry.Name())
		if !ok {
			continue
		}
		path := filepath.Join(archiveRoot, entry.Name())
		out = append(out, TreeNode{
			ID:       path,
			Name:     displayName,
			ClientID: clientID,
			Type:     "archived-client",
			Path:     path,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.ToLower(out[i].Name)
		right := strings.ToLower(out[j].Name)
		if left == right {
			return out[i].ClientID < out[j].ClientID
		}
		return left < right
	})
	return out, nil
}

func (a *App) UnarchiveClient(path string) (string, error) {
	if err := a.requireSignedIn(); err != nil {
		return "", err
	}
	path, err := a.safePath(path)
	if err != nil {
		return "", err
	}
	archiveRoot, _ := filepath.Abs(filepath.Join(a.dataRoot, ".Archive"))
	absPath, _ := filepath.Abs(path)
	if absPath == archiveRoot || !strings.HasPrefix(absPath, archiveRoot+string(os.PathSeparator)) {
		return "", errors.New("select an archived client first")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("archived client is not a folder")
	}
	clientID, displayName, ok := parseClientDirName(filepath.Base(absPath))
	if !ok {
		return "", errors.New("invalid archived client folder")
	}

	dest := filepath.Join(a.dataRoot, filepath.Base(absPath))
	if _, statErr := os.Stat(dest); !errors.Is(statErr, os.ErrNotExist) {
		// An archived client ID may have been reused while the client was archived.
		// Preserve the client name but assign the next available local client ID.
		newID := a.nextClientID()
		for {
			dest = filepath.Join(a.dataRoot, fmt.Sprintf("%05d - %s", newID, displayName))
			if _, statErr = os.Stat(dest); errors.Is(statErr, os.ErrNotExist) {
				clientID = fmt.Sprintf("%05d", newID)
				break
			}
			newID++
		}
	}
	if err := os.Rename(absPath, dest); err != nil {
		return "", err
	}
	a.remapDocumentMetadataPrefix(absPath, dest)
	_ = a.metadataRenamePrefix("client-profile", a.documentMetadataKey(absPath), a.documentMetadataKey(dest))
	_ = a.appendAudit(dest, "Client unarchived", fmt.Sprintf("Client restored to active workspace as ID %s", clientID))
	return dest, nil
}

func (a *App) ExitBabyFileCab() {
	_ = a.appendAppAuditCurrent("", "", "Application exited", "BabyFileCab was closed")
	wailsruntime.Quit(a.ctx)
}

func (a *App) GetDocumentProperties(path string) (DocumentProperties, error) {
	path, err := a.requireType(path, "file")
	if err != nil {
		return DocumentProperties{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return DocumentProperties{}, err
	}
	clientPath, err := a.clientPathFor(path)
	if err != nil {
		return DocumentProperties{}, err
	}
	clientID, clientName, _ := parseClientDirName(filepath.Base(clientPath))
	location, _ := filepath.Rel(clientPath, filepath.Dir(path))
	if location == "." {
		location = "Client root"
	}
	meta, _ := a.readDocumentMetadata()
	return DocumentProperties{
		Name:        filepath.Base(path),
		Path:        path,
		ClientName:  clientName,
		ClientID:    clientID,
		Location:    location,
		Description: meta[a.documentMetadataKey(path)],
		Size:        info.Size(),
		Modified:    info.ModTime().Format(time.RFC3339),
	}, nil
}

func (a *App) SaveDocumentDescription(path, description string) error {
	path, err := a.requireType(path, "file")
	if err != nil {
		return err
	}
	meta, err := a.readDocumentMetadata()
	if err != nil {
		return err
	}
	key := a.documentMetadataKey(path)
	description = strings.TrimSpace(description)
	if description == "" {
		delete(meta, key)
	} else {
		meta[key] = description
	}
	if err := a.writeDocumentMetadata(meta); err != nil {
		return err
	}
	_ = a.appendAudit(path, "Document description updated", filepath.Base(path))
	return nil
}

func (a *App) GlobalFind(query string) ([]GlobalSearchResult, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []GlobalSearchResult{}, nil
	}
	q := strings.ToLower(query)
	meta, _ := a.readDocumentMetadata()
	results := make([]GlobalSearchResult, 0)
	err := filepath.WalkDir(a.dataRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			// Legacy-company data may live at ~/BabyFileCabData. Never descend
			// into the container used by other company workspaces.
			if path != a.dataRoot && filepath.Dir(path) == a.dataRoot && d.Name() == "companies" {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if name == clientProfileFile || name == auditFile || name == richNotesFile || name == documentMetadataFile {
			return nil
		}
		clientPath, err := a.clientPathFor(path)
		if err != nil {
			return nil
		}
		clientID, clientName, ok := parseClientDirName(filepath.Base(clientPath))
		if !ok {
			return nil
		}
		desc := meta[a.documentMetadataKey(path)]
		rel, _ := filepath.Rel(clientPath, path)
		noteText := ""
		if name == notesFile {
			if data, readErr := a.secureReadFile(path); readErr == nil {
				noteText = string(data)
			}
		}
		hay := strings.ToLower(strings.Join([]string{name, desc, clientName, clientID, rel, noteText}, " "))
		if !strings.Contains(hay, q) {
			return nil
		}
		match := "File name"
		if strings.Contains(strings.ToLower(noteText), q) {
			match = "Client notes"
		} else if strings.Contains(strings.ToLower(desc), q) {
			match = "Description"
		} else if strings.Contains(strings.ToLower(clientName), q) || strings.Contains(clientID, q) {
			match = "Client"
		} else if strings.Contains(strings.ToLower(rel), q) {
			match = "Location"
		}
		results = append(results, GlobalSearchResult{
			Name: name, Path: path, ClientName: clientName, ClientID: clientID,
			Location: rel, Description: desc, Match: match,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool {
		if strings.ToLower(results[i].ClientName) == strings.ToLower(results[j].ClientName) {
			return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
		}
		return strings.ToLower(results[i].ClientName) < strings.ToLower(results[j].ClientName)
	})
	return results, nil
}

func (a *App) GetTree() ([]TreeNode, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(a.dataRoot)
	if err != nil {
		return nil, err
	}
	var clients []TreeNode
	for _, e := range entries {
		if !e.IsDir() || !isClientDirName(e.Name()) {
			continue
		}
		p := filepath.Join(a.dataRoot, e.Name())
		n, err := a.buildClientNode(p)
		if err == nil {
			clients = append(clients, n)
		}
	}
	sort.Slice(clients, func(i, j int) bool { return strings.ToLower(clients[i].Name) < strings.ToLower(clients[j].Name) })
	return clients, nil
}

func (a *App) buildClientNode(clientPath string) (TreeNode, error) {
	clientPath, err := a.safePath(clientPath)
	if err != nil {
		return TreeNode{}, err
	}
	base := filepath.Base(clientPath)
	clientID, displayName, ok := parseClientDirName(base)
	if !ok {
		return TreeNode{}, errors.New("invalid client folder")
	}
	// Keep the 5-digit client ID in the local folder name and node metadata,
	// but show only the human-friendly client name in the document tree.
	n := TreeNode{ID: clientPath, Name: displayName, ClientID: clientID, Type: "client", Path: clientPath}

	permanent := filepath.Join(clientPath, permanentFolder)
	if err := os.MkdirAll(permanent, 0700); err != nil {
		return TreeNode{}, err
	}
	notes := filepath.Join(permanent, notesFile)
	if _, err := os.Stat(notes); errors.Is(err, os.ErrNotExist) {
		if err := a.secureWriteFile(notes, []byte("")); err != nil {
			return TreeNode{}, err
		}
	}

	permNode, err := a.buildFolderNode(permanent, "permanent")
	if err != nil {
		return TreeNode{}, err
	}
	n.Children = append(n.Children, permNode)

	entries, err := os.ReadDir(clientPath)
	if err != nil {
		return TreeNode{}, err
	}
	var years, clientSections []TreeNode
	for _, e := range entries {
		if !e.IsDir() || e.Name() == permanentFolder || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(clientPath, e.Name())
		if isTaxYear(e.Name()) {
			yn, err := a.buildTaxYearNode(p)
			if err == nil {
				years = append(years, yn)
			}
		} else {
			sn, err := a.buildFolderNode(p, "section")
			if err == nil {
				clientSections = append(clientSections, sn)
			}
		}
	}
	sort.Slice(years, func(i, j int) bool { return years[i].Name > years[j].Name })
	sortNodes(clientSections)
	n.Children = append(n.Children, years...)
	n.Children = append(n.Children, clientSections...)
	return n, nil
}

func (a *App) buildTaxYearNode(path string) (TreeNode, error) {
	path, err := a.safePath(path)
	if err != nil {
		return TreeNode{}, err
	}
	n := TreeNode{ID: path, Name: filepath.Base(path), Type: "taxyear", Path: path}
	entries, err := os.ReadDir(path)
	if err != nil {
		return TreeNode{}, err
	}
	var dirs, files []TreeNode
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(path, e.Name())
		if e.IsDir() {
			cn, err := a.buildFolderNode(p, "section")
			if err == nil {
				dirs = append(dirs, cn)
			}
		} else {
			files = append(files, fileNode(p))
		}
	}
	sortNodes(dirs)
	sortNodes(files)
	n.Children = append(n.Children, dirs...)
	n.Children = append(n.Children, files...)
	return n, nil
}

func (a *App) buildFolderNode(path, nodeType string) (TreeNode, error) {
	path, err := a.safePath(path)
	if err != nil {
		return TreeNode{}, err
	}
	n := TreeNode{ID: path, Name: filepath.Base(path), Type: nodeType, Path: path}
	entries, err := os.ReadDir(path)
	if err != nil {
		return TreeNode{}, err
	}
	var dirs, files []TreeNode
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(path, e.Name())
		if e.IsDir() {
			cn, err := a.buildFolderNode(p, "section")
			if err == nil {
				dirs = append(dirs, cn)
			}
		} else {
			files = append(files, fileNode(p))
		}
	}
	sortNodes(dirs)
	sortNodes(files)
	// notes.txt should always be first in Permanent Folder.
	if nodeType == "permanent" {
		sort.SliceStable(files, func(i, j int) bool {
			if files[i].Name == notesFile {
				return true
			}
			if files[j].Name == notesFile {
				return false
			}
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})
	}
	n.Children = append(n.Children, dirs...)
	n.Children = append(n.Children, files...)
	return n, nil
}

func fileNode(path string) TreeNode {
	return TreeNode{ID: path, Name: filepath.Base(path), Type: "file", Path: path}
}

func sortNodes(nodes []TreeNode) {
	sort.Slice(nodes, func(i, j int) bool { return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name) })
}

func (a *App) AddClient(name, address, phone, email, taxID, returnType string) (string, error) {
	if err := a.requireSignedIn(); err != nil {
		return "", err
	}
	name = sanitizeName(name)
	if name == "" {
		return "", errors.New("client name is required")
	}
	returnType = strings.ToUpper(strings.TrimSpace(returnType))
	switch returnType {
	case "1040", "1065", "1120S", "1120":
	default:
		return "", errors.New("select a return type: 1040, 1065, 1120S, or 1120")
	}
	id := a.nextClientID()
	if id > 99999 {
		return "", errors.New("no 5-digit client IDs are available")
	}
	clientDir := filepath.Join(a.dataRoot, fmt.Sprintf("%05d - %s", id, name))
	if err := os.MkdirAll(filepath.Join(clientDir, permanentFolder), 0700); err != nil {
		return "", err
	}
	if err := a.secureWriteFile(filepath.Join(clientDir, permanentFolder, notesFile), []byte("")); err != nil {
		return "", err
	}
	profile := ClientProfile{
		Address:    strings.TrimSpace(address),
		Phone:      strings.TrimSpace(phone),
		Email:      strings.TrimSpace(email),
		TaxID:      strings.TrimSpace(taxID),
		ReturnType: returnType,
	}
	if err := a.writeClientProfile(clientDir, profile); err != nil {
		return "", err
	}
	_ = a.appendAudit(clientDir, "Client created", fmt.Sprintf("Client ID %05d • Return type %s", id, returnType))
	return clientDir, nil
}

func (a *App) RenameClient(path, newName string) (string, error) {
	path, err := a.requireType(path, "client")
	if err != nil {
		return "", err
	}
	newName = sanitizeName(newName)
	if newName == "" {
		return "", errors.New("client name is required")
	}
	base := filepath.Base(path)
	parts := strings.SplitN(base, " - ", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid client folder")
	}
	oldName := parts[1]
	newPath := filepath.Join(filepath.Dir(path), parts[0]+" - "+newName)
	if err := os.Rename(path, newPath); err != nil {
		return "", err
	}
	a.remapDocumentMetadataPrefix(path, newPath)
	_ = a.metadataRenamePrefix("client-profile", a.documentMetadataKey(path), a.documentMetadataKey(newPath))
	_ = a.appendAudit(newPath, "Client renamed", fmt.Sprintf("%s → %s", oldName, newName))
	return newPath, nil
}

func (a *App) DeleteClient(path string) error {
	path, err := a.requireType(path, "client")
	if err != nil {
		return err
	}
	_, displayName, _ := parseClientDirName(filepath.Base(path))
	_ = a.appendAudit(path, "Client deleted", displayName)
	a.removeDocumentMetadataPrefix(path)
	_ = a.metadataDelete("client-profile", a.documentMetadataKey(path))
	return os.RemoveAll(path)
}

func (a *App) AddTaxYear(clientPath, year string) (string, error) {
	clientPath, err := a.requireType(clientPath, "client")
	if err != nil {
		return "", err
	}
	year = strings.TrimSpace(year)
	if !isTaxYear(year) {
		return "", errors.New("enter a 4-digit tax year, for example 2026")
	}
	p := filepath.Join(clientPath, year)
	if err := os.Mkdir(p, 0700); err != nil {
		return "", err
	}
	_ = a.appendAudit(p, "Tax year added", year)
	return p, nil
}

func (a *App) RenameTaxYear(path, year string) (string, error) {
	path, err := a.requireType(path, "taxyear")
	if err != nil {
		return "", err
	}
	oldYear := filepath.Base(path)
	year = strings.TrimSpace(year)
	if !isTaxYear(year) {
		return "", errors.New("enter a 4-digit tax year")
	}
	newPath := filepath.Join(filepath.Dir(path), year)
	if err := os.Rename(path, newPath); err != nil {
		return "", err
	}
	a.remapDocumentMetadataPrefix(path, newPath)
	_ = a.appendAudit(newPath, "Tax year renamed", fmt.Sprintf("%s → %s", oldYear, year))
	return newPath, nil
}

func (a *App) DeleteTaxYear(path string) error {
	path, err := a.requireType(path, "taxyear")
	if err != nil {
		return err
	}
	_ = a.appendAudit(path, "Tax year deleted", filepath.Base(path))
	a.removeDocumentMetadataPrefix(path)
	return os.RemoveAll(path)
}

func (a *App) AddSection(parentPath, name string) (string, error) {
	parentPath, err := a.safePath(parentPath)
	if err != nil {
		return "", err
	}
	t := a.pathType(parentPath)
	if t != "client" && t != "taxyear" && t != "section" {
		return "", errors.New("select a client, tax year, or section first")
	}
	// A section selected for Add Section creates a sibling, never nests or moves documents.
	if t == "section" {
		parentPath = filepath.Dir(parentPath)
	}
	name = sanitizeName(name)
	if name == "" {
		return "", errors.New("section name is required")
	}
	if isTaxYear(name) || strings.HasPrefix(name, ".") || name == permanentFolder {
		return "", errors.New("choose a section name other than a tax year or reserved folder name")
	}
	p := filepath.Join(parentPath, name)
	if err := os.Mkdir(p, 0700); err != nil {
		return "", err
	}
	_ = a.appendAudit(p, "Section added", name)
	return p, nil
}

func (a *App) RenameSection(path, name string) (string, error) {
	path, err := a.requireType(path, "section")
	if err != nil {
		return "", err
	}
	name = sanitizeName(name)
	if name == "" {
		return "", errors.New("section name is required")
	}
	oldName := filepath.Base(path)
	newPath := filepath.Join(filepath.Dir(path), name)
	if err := os.Rename(path, newPath); err != nil {
		return "", err
	}
	a.remapDocumentMetadataPrefix(path, newPath)
	_ = a.appendAudit(newPath, "Section renamed", fmt.Sprintf("%s → %s", oldName, name))
	return newPath, nil
}

func (a *App) DeleteSection(path string) error {
	path, err := a.requireType(path, "section")
	if err != nil {
		return err
	}
	// Remove only an empty directory; os.Remove also closes the check/delete race.
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("delete or move every file and subsection before deleting this section")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("section must be empty before deletion: %w", err)
	}
	_ = a.appendAudit(path, "Section deleted", filepath.Base(path))
	a.removeDocumentMetadataPrefix(path)
	return nil
}

func (a *App) UploadFiles(targetPath string) ([]string, error) {
	if a.ctx == nil {
		return nil, errors.New("application is not ready")
	}
	targetPath, err := a.uploadTarget(targetPath)
	if err != nil {
		return nil, err
	}
	files, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Upload Files to BabyFileCab",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Office & Email Documents", Pattern: "*.pdf;*.doc;*.docx;*.rtf;*.xls;*.xlsx;*.xlsm;*.csv;*.eml;*.msg;*.mbox;*.oft;*.png;*.jpg;*.jpeg;*.gif;*.webp;*.txt"},
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
			{DisplayName: "Word Documents", Pattern: "*.doc;*.docx;*.rtf"},
			{DisplayName: "Excel Documents", Pattern: "*.xls;*.xlsx;*.xlsm;*.csv"},
			{DisplayName: "Email Files", Pattern: "*.eml;*.msg;*.mbox;*.oft"},
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
	if err != nil {
		return nil, err
	}
	var copied []string
	for _, src := range files {
		if src == "" {
			continue
		}
		dst := uniqueDestination(filepath.Join(targetPath, filepath.Base(src)))
		if err := a.secureCopyFile(src, dst); err != nil {
			return copied, err
		}
		copied = append(copied, dst)
	}
	if len(copied) > 0 {
		names := make([]string, 0, len(copied))
		for _, p := range copied {
			names = append(names, filepath.Base(p))
		}
		_ = a.appendAudit(targetPath, "Files uploaded", strings.Join(names, ", "))
	}
	return copied, nil
}

func (a *App) MoveDocument(sourcePath, targetPath string) (string, error) {
	sourcePath, err := a.requireType(sourcePath, "file")
	if err != nil {
		return "", err
	}
	if filepath.Base(sourcePath) == notesFile {
		return "", errors.New("notes.txt is protected and cannot be moved")
	}

	targetDir, err := a.uploadTarget(targetPath)
	if err != nil {
		return "", err
	}
	targetDir, err = a.safePath(targetDir)
	if err != nil {
		return "", err
	}

	// Dropping back into the same folder is a no-op.
	if filepath.Clean(filepath.Dir(sourcePath)) == filepath.Clean(targetDir) {
		return sourcePath, nil
	}

	sourceClient, err := a.clientPathFor(sourcePath)
	if err != nil {
		return "", err
	}
	destClient, err := a.clientPathFor(targetDir)
	if err != nil {
		return "", err
	}

	fileName := filepath.Base(sourcePath)
	destPath := uniqueDestination(filepath.Join(targetDir, fileName))
	if err := os.Rename(sourcePath, destPath); err != nil {
		return "", err
	}
	a.moveDocumentMetadata(sourcePath, destPath)

	// Record the move in the destination client's audit trail. If the move
	// crosses clients, also leave a history entry in the source client.
	destLabel := relativeClientLocation(destClient, destPath)
	sourceLabel := relativeClientLocation(sourceClient, sourcePath)
	if filepath.Clean(sourceClient) == filepath.Clean(destClient) {
		_ = a.appendAudit(destPath, "Document moved", fmt.Sprintf("%s: %s → %s", filepath.Base(destPath), sourceLabel, destLabel))
	} else {
		srcID, srcName, _ := parseClientDirName(filepath.Base(sourceClient))
		dstID, dstName, _ := parseClientDirName(filepath.Base(destClient))
		_ = a.appendAudit(sourceClient, "Document moved out", fmt.Sprintf("%s → %s (%s), %s", fileName, dstName, dstID, destLabel))
		_ = a.appendAudit(destPath, "Document moved in", fmt.Sprintf("%s ← %s (%s), %s", filepath.Base(destPath), srcName, srcID, sourceLabel))
	}

	return destPath, nil
}

func relativeClientLocation(clientPath, itemPath string) string {
	rel, err := filepath.Rel(clientPath, itemPath)
	if err != nil || rel == "." {
		return filepath.Base(itemPath)
	}
	return rel
}

func (a *App) ImportDroppedFiles(targetPath string, paths []string) ([]string, error) {
	targetPath, err := a.uploadTarget(targetPath)
	if err != nil {
		return nil, err
	}
	var copied []string
	for _, src := range paths {
		src = strings.TrimSpace(src)
		if src == "" {
			continue
		}
		info, err := os.Stat(src)
		if err != nil || info.IsDir() {
			continue
		}
		dst := uniqueDestination(filepath.Join(targetPath, filepath.Base(src)))
		if err := a.secureCopyFile(src, dst); err != nil {
			return copied, err
		}
		copied = append(copied, dst)
	}
	if len(copied) == 0 {
		return nil, errors.New("no files were dropped")
	}
	names := make([]string, 0, len(copied))
	for _, p := range copied {
		names = append(names, filepath.Base(p))
	}
	_ = a.appendAudit(targetPath, "Files added by drag and drop", strings.Join(names, ", "))
	return copied, nil
}

func (a *App) RenameFile(path, newName string) (string, error) {
	path, err := a.requireType(path, "file")
	if err != nil {
		return "", err
	}
	if filepath.Base(path) == notesFile {
		return "", errors.New("notes.txt is protected")
	}
	newName = strings.TrimSpace(filepath.Base(newName))
	if newName == "" {
		return "", errors.New("file name is required")
	}
	oldName := filepath.Base(path)
	newPath := filepath.Join(filepath.Dir(path), newName)
	if newPath == path {
		return path, nil
	}
	if _, err := os.Lstat(newPath); err == nil {
		return "", errors.New("a file with that name already exists in this section; choose another name")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.Rename(path, newPath); err != nil {
		return "", err
	}
	a.moveDocumentMetadata(path, newPath)
	_ = a.appendAudit(newPath, "File renamed", fmt.Sprintf("%s → %s", oldName, newName))
	return newPath, nil
}

func (a *App) DeleteFile(path string) error {
	path, err := a.requireType(path, "file")
	if err != nil {
		return err
	}
	if filepath.Base(path) == notesFile {
		return errors.New("notes.txt is protected")
	}
	name := filepath.Base(path)
	_ = a.appendAudit(path, "File deleted", name)
	a.removeDocumentMetadata(path)
	return os.Remove(path)
}

func (a *App) GetNotes(path string) (string, error) {
	np, err := a.notesPathFor(path)
	if err != nil {
		return "", err
	}
	b, err := a.secureReadFile(np)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (a *App) GetNotesDocument(path string) (NotesDocument, error) {
	np, err := a.notesPathFor(path)
	if err != nil {
		return NotesDocument{}, err
	}
	b, err := a.secureReadFile(np)
	if err != nil {
		return NotesDocument{}, err
	}
	doc := NotesDocument{Text: string(b)}
	rp, err := a.richNotesPathFor(path)
	if err == nil {
		if hb, readErr := a.secureReadFile(rp); readErr == nil {
			clean, sanitizeErr := sanitizeNotesHTML(string(hb))
            if sanitizeErr != nil { return NotesDocument{}, sanitizeErr }
            doc.HTML = clean
		}
	}
	return doc, nil
}

func (a *App) SaveNotes(path, text string) error {
	np, err := a.notesPathFor(path)
	if err != nil {
		return err
	}
	if err := a.secureWriteFile(np, []byte(text)); err != nil {
		return err
	}
	_ = a.appendAudit(path, "Client notes saved", "notes.txt updated")
	return nil
}

func (a *App) SaveNotesDocument(path, htmlContent, text string) error {
    cleanHTML, err := sanitizeNotesHTML(htmlContent)
    if err != nil { return err }
	np, err := a.notesPathFor(path)
	if err != nil {
		return err
	}
	rp, err := a.richNotesPathFor(path)
	if err != nil {
		return err
	}
	if err := a.secureWriteFile(np, []byte(text)); err != nil {
		return err
	}
	if err := a.secureWriteFile(rp, []byte(cleanHTML)); err != nil {
		return err
	}
	_ = a.appendAudit(path, "Client notes saved", "Formatted notes and notes.txt updated")
	return nil
}

func (a *App) GetAuditTrail(path string) ([]AuditEntry, error) {
	cp, err := a.clientPathFor(path)
	if err != nil {
		return nil, err
	}
	clientID, _, _ := parseClientDirName(filepath.Base(cp))
	entries := make([]AuditEntry, 0)
	if err := a.loadAuditSQLite("client", clientID, func(b []byte) error {
		var entry AuditEntry
		if err := json.Unmarshal(b, &entry); err != nil {
			return err
		}
		entries = append(entries, entry)
		return nil
	}); err != nil {
		return nil, err
	}
	sortClientAudit(entries)
	return entries, nil
}

func (a *App) GetPreview(path string) (Preview, error) {
	path, err := a.requireType(path, "file")
	if err != nil {
		return Preview{}, err
	}
	name := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))
	p := Preview{Name: name, Path: path}
	previewPath, cleanup, err := a.decryptedTemp(path)
	if err != nil {
		return p, err
	}
	defer cleanup()

	if name == notesFile || ext == ".txt" {
		b, err := a.secureReadFile(path)
		if err != nil {
			return p, err
		}
		p.Kind = "text"
		p.Text = string(b)
		return p, nil
	}
	if ext == ".csv" {
		rows, err := readCSV(previewPath, 200)
		if err != nil {
			return p, err
		}
		p.Kind = "csv"
		p.Rows = rows
		return p, nil
	}
	if ext == ".xlsx" || ext == ".xlsm" {
		rows, err := readXLSX(previewPath, 200, 60)
		if err == nil && len(rows) > 0 {
			p.Kind = "spreadsheet"
			p.Rows = rows
			return p, nil
		}
	}
	if ext == ".eml" {
		text, err := emlText(previewPath)
		if err == nil {
			p.Kind = "email"
			p.Text = text
			return p, nil
		}
	}
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
		b, err := a.secureReadFile(path)
		if err != nil {
			return p, err
		}
		mt := mime.TypeByExtension(ext)
		if mt == "" {
			mt = "image/png"
		}
		p.Kind = "image"
		p.ImageData = "data:" + mt + ";base64," + base64.StdEncoding.EncodeToString(b)
		return p, nil
	}
	if ext == ".pdf" {
		count, err := pdfPageCount(previewPath)
		if err != nil {
			p.Kind = "unsupported"
			p.Message = "PDF preview requires poppler-utils (pdfinfo and pdftoppm). The file is stored safely and can still be opened externally."
			return p, nil
		}
		p.Kind = "pdf"
		p.PageCount = count
		return p, nil
	}
	if ext == ".docx" {
		text, err := docxText(previewPath)
		if err == nil && strings.TrimSpace(text) != "" {
			p.Kind = "text"
			p.Text = text
			return p, nil
		}
	}
	p.Kind = "unsupported"
	p.Message = "This file is stored in BabyFileCab. Legacy .doc/.xls and Outlook .msg/.oft files can be opened in their normal desktop application."
	return p, nil
}

func (a *App) RenderPDFPage(path string, page int) (string, error) {
	path, err := a.requireType(path, "file")
	if err != nil {
		return "", err
	}
	if strings.ToLower(filepath.Ext(path)) != ".pdf" {
		return "", errors.New("not a PDF file")
	}
	plainPath, cleanup, err := a.decryptedTemp(path)
	if err != nil {
		return "", err
	}
	defer cleanup()
	if page < 1 {
		page = 1
	}
	tmpDir, err := os.MkdirTemp(filepath.Join(a.storageRoot, ".plaintext-preview"), "pdf-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	out := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-f", strconv.Itoa(page), "-singlefile", "-png", "-r", "130", plainPath, out)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("PDF preview failed: %v: %s", err, strings.TrimSpace(string(output)))
	}
	b, err := os.ReadFile(out + ".png")
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b), nil
}

func (a *App) SearchPDF(path, query string) (PDFSearchResponse, error) {
	path, err := a.requireType(path, "file")
	if err != nil {
		return PDFSearchResponse{}, err
	}
	if strings.ToLower(filepath.Ext(path)) != ".pdf" {
		return PDFSearchResponse{}, errors.New("select a PDF file first")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return PDFSearchResponse{}, errors.New("enter a word or phrase to search")
	}
	plainPath, cleanup, err := a.decryptedTemp(path)
	if err != nil {
		return PDFSearchResponse{}, err
	}
	defer cleanup()

	// -bbox gives us every searchable PDF word plus its exact page coordinates.
	// This is more reliable than splitting the normal pdftotext output on page-break
	// characters, and it lets the frontend highlight the actual match on the page.
	cmd := exec.Command("pdftotext", "-bbox", "-enc", "UTF-8", plainPath, "-")
	out, err := cmd.Output()
	if err != nil {
		detail := ""
		if ee, ok := err.(*exec.ExitError); ok {
			detail = strings.TrimSpace(string(ee.Stderr))
		}
		if detail != "" {
			return PDFSearchResponse{}, fmt.Errorf("PDF search failed: %v: %s", err, detail)
		}
		return PDFSearchResponse{}, fmt.Errorf("PDF search requires poppler-utils (pdftotext): %v", err)
	}

	var doc pdfBBoxDocument
	if err := xml.Unmarshal(out, &doc); err != nil {
		return PDFSearchResponse{}, fmt.Errorf("could not read searchable PDF text: %w", err)
	}

	queryWords := normalizePDFSearchQuery(query)
	if len(queryWords) == 0 {
		return PDFSearchResponse{}, errors.New("enter a searchable word or phrase")
	}

	resp := PDFSearchResponse{Query: query}
	const maxReturnedMatches = 500
	totalWords := 0

	for pageIndex, page := range doc.Pages {
		if len(page.Words) == 0 {
			continue
		}
		totalWords += len(page.Words)
		normalized := make([]string, len(page.Words))
		for i, word := range page.Words {
			normalized[i] = normalizePDFSearchToken(word.Text)
		}

		for i := 0; i+len(queryWords) <= len(page.Words); i++ {
			matched := true
			for j, q := range queryWords {
				// Match like a normal Ctrl+F search: case-insensitive and
				// able to find a typed fragment inside a PDF word.
				if q == "" || !strings.Contains(normalized[i+j], q) {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}

			resp.Total++
			if len(resp.Results) >= maxReturnedMatches {
				continue
			}

			first := page.Words[i]
			xMin, yMin, xMax, yMax := first.XMin, first.YMin, first.XMax, first.YMax
			for _, w := range page.Words[i : i+len(queryWords)] {
				if w.XMin < xMin {
					xMin = w.XMin
				}
				if w.YMin < yMin {
					yMin = w.YMin
				}
				if w.XMax > xMax {
					xMax = w.XMax
				}
				if w.YMax > yMax {
					yMax = w.YMax
				}
			}

			resp.Results = append(resp.Results, PDFSearchMatch{
				Page:       pageIndex + 1,
				Snippet:    pdfWordSnippet(page.Words, i, len(queryWords)),
				X:          xMin,
				Y:          yMin,
				Width:      xMax - xMin,
				Height:     yMax - yMin,
				PageWidth:  page.Width,
				PageHeight: page.Height,
			})
		}
	}

	resp.Searchable = totalWords > 0
	if !resp.Searchable {
		resp.Message = "This PDF has no searchable text. It may be a scanned/image-only PDF."
	} else if resp.Total == 0 {
		resp.Message = fmt.Sprintf("No match for %q was found in this PDF.", query)
	}
	return resp, nil
}

func normalizePDFSearchQuery(query string) []string {
	fields := strings.Fields(query)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if token := normalizePDFSearchToken(field); token != "" {
			out = append(out, token)
		}
	}
	return out
}

func normalizePDFSearchToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return s
}

func pdfWordSnippet(words []pdfBBoxWord, matchStart, matchCount int) string {
	start := matchStart - 8
	if start < 0 {
		start = 0
	}
	end := matchStart + matchCount + 10
	if end > len(words) {
		end = len(words)
	}
	parts := make([]string, 0, end-start)
	for _, word := range words[start:end] {
		parts = append(parts, strings.TrimSpace(word.Text))
	}
	snippet := strings.Join(parts, " ")
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(words) {
		snippet += "…"
	}
	return snippet
}

func (a *App) OpenFile(path string) error {
	path, err := a.safePath(path)
	if err != nil {
		return err
	}
	plainPath, cleanup, err := a.decryptedTemp(path)
	if err != nil {
		return err
	}
	if err := openPath(plainPath); err != nil {
		cleanup()
		return err
	}
	// External viewers may read asynchronously; cleanup occurs on lock/sign out.
	return nil
}

func (a *App) OpenIRSPaymentPage() error {
	if err := a.requireSignedIn(); err != nil {
		return err
	}
	wailsruntime.BrowserOpenURL(a.ctx, "https://www.irs.gov/payments")
	_ = a.appendAppAuditCurrent("", "", "IRS payment website opened", "https://www.irs.gov/payments")
	return nil
}

func (a *App) OpenIRSCurrencyExchangeRatesPage() error {
	if err := a.requireSignedIn(); err != nil {
		return err
	}
	const url = "https://www.irs.gov/individuals/international-taxpayers/yearly-average-currency-exchange-rates"
	wailsruntime.BrowserOpenURL(a.ctx, url)
	_ = a.appendAppAuditCurrent("", "", "IRS currency exchange-rate website opened", url)
	return nil
}

func (a *App) engagementLetterLogoPath() string {
	return filepath.Join(a.dataRoot, engagementLogoFile)
}

func (a *App) GetEngagementLetterLogo() (EngagementLetterLogoInfo, error) {
	if err := a.requireSignedIn(); err != nil {
		return EngagementLetterLogoInfo{}, err
	}
	path := a.engagementLetterLogoPath()
	data, err := a.secureReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return EngagementLetterLogoInfo{}, nil
	}
	if err != nil {
		return EngagementLetterLogoInfo{}, err
	}
	return EngagementLetterLogoInfo{
		Path:    path,
		DataURL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

func (a *App) ChooseEngagementLetterLogo() (EngagementLetterLogoInfo, error) {
	if err := a.requireSignedIn(); err != nil {
		return EngagementLetterLogoInfo{}, err
	}
	chosen, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Choose Firm Logo",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Image Files (*.png;*.jpg;*.jpeg)", Pattern: "*.png;*.jpg;*.jpeg"},
		},
	})
	if err != nil {
		return EngagementLetterLogoInfo{}, err
	}
	if strings.TrimSpace(chosen) == "" {
		return a.GetEngagementLetterLogo()
	}
	info, err := os.Stat(chosen)
	if err != nil {
		return EngagementLetterLogoInfo{}, err
	}
	if info.Size() > 10*1024*1024 {
		return EngagementLetterLogoInfo{}, errors.New("firm logo must be 10 MB or smaller")
	}
	file, err := os.Open(chosen)
	if err != nil {
		return EngagementLetterLogoInfo{}, err
	}
	img, _, decodeErr := image.Decode(file)
	_ = file.Close()
	if decodeErr != nil {
		return EngagementLetterLogoInfo{}, errors.New("choose a valid PNG or JPEG logo")
	}
	bounds := img.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 || bounds.Dx() > 10000 || bounds.Dy() > 10000 {
		return EngagementLetterLogoInfo{}, errors.New("firm logo dimensions are not supported")
	}
	// Normalize all logos to JPEG so PDF and Word generation can embed one
	// predictable image format. Transparent PNG areas are placed on white.
	canvas := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), img, bounds.Min, draw.Over)

	path := a.engagementLetterLogoPath()
	var logoBytes bytes.Buffer
	if err := jpeg.Encode(&logoBytes, canvas, &jpeg.Options{Quality: 92}); err != nil {
		return EngagementLetterLogoInfo{}, err
	}
	if err := a.secureWriteFile(path, logoBytes.Bytes()); err != nil {
		return EngagementLetterLogoInfo{}, err
	}
	_ = a.appendAppAuditCurrent("", "", "Engagement Letter logo updated", "Firm logo saved for the current company account")
	return a.GetEngagementLetterLogo()
}

func (a *App) ClearEngagementLetterLogo() error {
	if err := a.requireSignedIn(); err != nil {
		return err
	}
	path := a.engagementLetterLogoPath()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_ = a.appendAppAuditCurrent("", "", "Engagement Letter logo removed", "Firm logo removed for the current company account")
	return nil
}

func (a *App) EngagementLetterDefaultOutput(clientPath, service, taxYear string) (string, error) {
	if err := a.requireSignedIn(); err != nil {
		return "", err
	}
	props, err := a.GetClientProperties(clientPath)
	if err != nil {
		return "", err
	}
	folder := filepath.Join(a.dataRoot, "Generated Forms", "Engagement Letters")
	if err := os.MkdirAll(folder, 0700); err != nil {
		return "", err
	}
	name := engagementSafeFilename(props.Name)
	service = engagementSafeFilename(service)
	taxYear = engagementSafeFilename(taxYear)
	if service == "" {
		service = "Service"
	}
	if taxYear == "" {
		taxYear = "Year"
	}
	return filepath.Join(folder, fmt.Sprintf("Engagement_Letter_%s_%s_%s.pdf", name, service, taxYear)), nil
}

func (a *App) ChooseEngagementLetterSavePath(outputKind, clientPath, service, taxYear string) (string, error) {
	if err := a.requireSignedIn(); err != nil {
		return "", err
	}
	outputKind = strings.ToLower(strings.TrimSpace(outputKind))
	if outputKind != "pdf" && outputKind != "docx" {
		return "", errors.New("choose PDF or Microsoft Word")
	}
	defaultPDF, err := a.EngagementLetterDefaultOutput(clientPath, service, taxYear)
	if err != nil {
		return "", err
	}
	stem := strings.TrimSuffix(defaultPDF, filepath.Ext(defaultPDF))
	ext := ".pdf"
	title := "Save PDF Engagement Letter"
	filter := wailsruntime.FileFilter{DisplayName: "PDF Document (*.pdf)", Pattern: "*.pdf"}
	if outputKind == "docx" {
		ext = ".docx"
		title = "Save Microsoft Word Engagement Letter"
		filter = wailsruntime.FileFilter{DisplayName: "Microsoft Word Document (*.docx)", Pattern: "*.docx"}
	}
	chosen, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: filepath.Base(stem + ext),
		Filters:         []wailsruntime.FileFilter{filter},
	})
	if err != nil {
		return "", err
	}
	chosen = strings.TrimSpace(chosen)
	if chosen == "" {
		return "", nil
	}
	if !strings.EqualFold(filepath.Ext(chosen), ext) {
		chosen = strings.TrimSuffix(chosen, filepath.Ext(chosen)) + ext
	}
	return chosen, nil
}

func (a *App) ChooseEngagementLetterOutput(suggestedPath string) (string, error) {
	if err := a.requireSignedIn(); err != nil {
		return "", err
	}
	defaultName := filepath.Base(strings.TrimSpace(suggestedPath))
	if defaultName == "." || defaultName == "" {
		defaultName = "Engagement_Letter.pdf"
	}
	if !strings.EqualFold(filepath.Ext(defaultName), ".pdf") {
		defaultName = strings.TrimSuffix(defaultName, filepath.Ext(defaultName)) + ".pdf"
	}
	chosen, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save Engagement Letter",
		DefaultFilename: defaultName,
		Filters:         []wailsruntime.FileFilter{{DisplayName: "PDF Document (*.pdf)", Pattern: "*.pdf"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(chosen) == "" {
		return "", nil
	}
	if !strings.EqualFold(filepath.Ext(chosen), ".pdf") {
		chosen += ".pdf"
	}
	return chosen, nil
}

func (a *App) GenerateEngagementLetter(req EngagementLetterRequest, outputKind string) ([]string, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	data, err := a.buildEngagementLetterData(req)
	if err != nil {
		return nil, err
	}
	if data.LogoPath != "" {
		logo, cleanup, err := a.decryptedTemp(data.LogoPath)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		data.LogoPath = logo
	}
	outputKind = strings.ToLower(strings.TrimSpace(outputKind))
	if outputKind != "pdf" && outputKind != "docx" && outputKind != "both" {
		return nil, errors.New("choose PDF, Word, or Both")
	}

	basePath := strings.TrimSpace(req.OutputPath)
	if basePath == "" {
		basePath, err = a.EngagementLetterDefaultOutput(data.ClientPath, data.Service, data.TaxYear)
		if err != nil {
			return nil, err
		}
	}
	basePath, err = filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}
	stem := strings.TrimSuffix(basePath, filepath.Ext(basePath))
	pdfPath := stem + ".pdf"
	docxPath := stem + ".docx"
	if err := os.MkdirAll(filepath.Dir(basePath), 0700); err != nil {
		return nil, err
	}

	generated := make([]string, 0, 2)
	if outputKind == "pdf" || outputKind == "both" {
		if err := a.generateOutput(pdfPath, func(p string) error { return writeEngagementLetterPDF(p, data) }); err != nil {
			return nil, fmt.Errorf("create engagement-letter PDF: %w", err)
		}
		generated = append(generated, pdfPath)
	}
	if outputKind == "docx" || outputKind == "both" {
		if err := a.generateOutput(docxPath, func(p string) error { return writeEngagementLetterDOCX(p, data) }); err != nil {
			return nil, fmt.Errorf("create engagement-letter Word document: %w", err)
		}
		generated = append(generated, docxPath)
	}

	detail := fmt.Sprintf("Service %s; tax year/period %s; %s %s; preparer %s; formats %s", data.Service, data.TaxYear, data.FeeType, data.FeeValue, data.PreparerName, outputKind)
	_ = a.appendAudit(data.ClientPath, "Engagement letter generated", detail)
	_ = a.appendAppAuditCurrent(data.ClientID, data.ClientName, "Engagement Letter generated", detail)
	for _, path := range generated {
		_ = a.openGeneratedOutput(path)
	}
	return generated, nil
}

func (a *App) ChooseInvoiceSavePath(outputKind, clientPath, invoiceNumber string) (string, error) {
	if err := a.requireSignedIn(); err != nil {
		return "", err
	}
	outputKind = strings.ToLower(strings.TrimSpace(outputKind))
	if outputKind != "pdf" && outputKind != "docx" {
		return "", errors.New("choose PDF or Microsoft Word")
	}
	props, err := a.GetClientProperties(clientPath)
	if err != nil {
		return "", err
	}
	folder := filepath.Join(a.dataRoot, "Generated Forms", "Invoices")
	if err := os.MkdirAll(folder, 0700); err != nil {
		return "", err
	}
	clientName := engagementSafeFilename(props.Name)
	invoiceSafe := engagementSafeFilename(invoiceNumber)
	if invoiceSafe == "Client" {
		invoiceSafe = "Invoice"
	}
	ext := ".pdf"
	title := "Save PDF Invoice"
	filter := wailsruntime.FileFilter{DisplayName: "PDF Document (*.pdf)", Pattern: "*.pdf"}
	if outputKind == "docx" {
		ext = ".docx"
		title = "Save Microsoft Word Invoice"
		filter = wailsruntime.FileFilter{DisplayName: "Microsoft Word Document (*.docx)", Pattern: "*.docx"}
	}
	defaultName := fmt.Sprintf("Invoice_%s_%s%s", clientName, invoiceSafe, ext)
	chosen, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:            title,
		DefaultDirectory: folder,
		DefaultFilename:  defaultName,
		Filters:          []wailsruntime.FileFilter{filter},
	})
	if err != nil {
		return "", err
	}
	chosen = strings.TrimSpace(chosen)
	if chosen == "" {
		return "", nil
	}
	if !strings.EqualFold(filepath.Ext(chosen), ext) {
		chosen = strings.TrimSuffix(chosen, filepath.Ext(chosen)) + ext
	}
	return chosen, nil
}

func (a *App) GenerateInvoice(req InvoiceRequest, outputKind string) ([]string, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	data, err := a.buildInvoiceData(req)
	if err != nil {
		return nil, err
	}
	if data.LogoPath != "" {
		logo, cleanup, err := a.decryptedTemp(data.LogoPath)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		data.LogoPath = logo
	}
	outputKind = strings.ToLower(strings.TrimSpace(outputKind))
	if outputKind != "pdf" && outputKind != "docx" {
		return nil, errors.New("choose PDF or Microsoft Word")
	}
	outputPath := strings.TrimSpace(req.OutputPath)
	if outputPath == "" {
		outputPath, err = a.ChooseInvoiceSavePath(outputKind, data.ClientPath, data.InvoiceNumber)
		if err != nil {
			return nil, err
		}
		if outputPath == "" {
			return nil, nil
		}
	}
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return nil, err
	}
	if outputKind == "pdf" {
		if !strings.EqualFold(filepath.Ext(outputPath), ".pdf") {
			outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".pdf"
		}
		if err := a.generateOutput(outputPath, func(p string) error { return writeInvoicePDF(p, data) }); err != nil {
			return nil, fmt.Errorf("create invoice PDF: %w", err)
		}
	} else {
		if !strings.EqualFold(filepath.Ext(outputPath), ".docx") {
			outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".docx"
		}
		if err := a.generateOutput(outputPath, func(p string) error { return writeInvoiceDOCX(p, data) }); err != nil {
			return nil, fmt.Errorf("create invoice Word document: %w", err)
		}
	}
	detail := fmt.Sprintf("Invoice %s; service %s; tax year %s; %s; total $%.2f; preparer %s; format %s", data.InvoiceNumber, data.Service, data.TaxYear, data.FeeType, data.Total, data.PreparerName, outputKind)
	_ = a.appendAudit(data.ClientPath, "Invoice generated", detail)
	_ = a.appendAppAuditCurrent(data.ClientID, data.ClientName, "Invoice generated", detail)
	_ = a.openGeneratedOutput(outputPath)
	return []string{outputPath}, nil
}

func (a *App) buildInvoiceData(req InvoiceRequest) (invoiceData, error) {
	props, err := a.GetClientProperties(req.ClientPath)
	if err != nil {
		return invoiceData{}, err
	}
	service := strings.TrimSpace(req.Service)
	validService := false
	for _, item := range []string{"1040", "1041", "1065", "1120S", "1120", "Bookkeeping", "Tax Projection"} {
		if service == item {
			validService = true
			break
		}
	}
	if !validService {
		return invoiceData{}, errors.New("select a valid invoice service")
	}
	taxYear := strings.TrimSpace(req.TaxYear)
	yearNumber, yearErr := strconv.Atoi(taxYear)
	if yearErr != nil || yearNumber < 2018 || yearNumber > 2026 {
		return invoiceData{}, errors.New("select a tax year from 2018 through 2026")
	}
	invoiceNumber := strings.TrimSpace(req.InvoiceNumber)
	if invoiceNumber == "" {
		return invoiceData{}, errors.New("enter an invoice number")
	}
	invoiceDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.InvoiceDate))
	if err != nil {
		return invoiceData{}, errors.New("choose a valid invoice date")
	}
	dueDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.DueDate))
	if err != nil {
		return invoiceData{}, errors.New("choose a valid due date")
	}
	if dueDate.Before(invoiceDate) {
		return invoiceData{}, errors.New("the due date cannot be earlier than the invoice date")
	}
	feeType := strings.TrimSpace(req.FeeType)
	if feeType != "Flat Fee" && feeType != "Hourly Rate" {
		return invoiceData{}, errors.New("select Flat Fee or Hourly Rate")
	}
	rate, err := invoiceParseNumber(req.FeeValue)
	if err != nil || rate <= 0 {
		return invoiceData{}, errors.New("enter a valid fee or hourly rate greater than zero")
	}
	hours := 1.0
	if feeType == "Hourly Rate" {
		hours, err = invoiceParseNumber(req.Hours)
		if err != nil || hours <= 0 {
			return invoiceData{}, errors.New("enter valid hours greater than zero")
		}
	}

	users, err := a.ListCompanyUsers()
	if err != nil {
		return invoiceData{}, err
	}
	var preparer *UserProfile
	for i := range users {
		if strings.EqualFold(users[i].Username, strings.TrimSpace(req.PreparerUsername)) {
			copy := users[i]
			preparer = &copy
			break
		}
	}
	if preparer == nil {
		return invoiceData{}, errors.New("select a preparer from the current company")
	}

	companyName := ""
	a.authMu.RLock()
	if a.currentCompany != nil {
		companyName = strings.TrimSpace(a.currentCompany.Name)
	}
	a.authMu.RUnlock()
	if companyName == "" {
		companyName = strings.TrimSpace(preparer.CompanyName)
	}
	preparerName := strings.TrimSpace(preparer.FirstName + " " + preparer.LastName)
	if preparerName == "" {
		preparerName = preparer.Username
	}
	preparerAddress := make([]string, 0, 2)
	if strings.TrimSpace(preparer.StreetAddress) != "" {
		preparerAddress = append(preparerAddress, strings.TrimSpace(preparer.StreetAddress))
	}
	if line := engagementCityStateZIP(*preparer); line != "" {
		preparerAddress = append(preparerAddress, line)
	}
	phone := strings.TrimSpace(preparer.Telephone)
	if phone == "" {
		phone = strings.TrimSpace(preparer.Phone)
	}
	description := strings.TrimSpace(req.Description)
	serviceDesc := engagementServiceDescription(service)
	if description == "" {
		description = fmt.Sprintf("%s - Tax Year %s", serviceDesc, taxYear)
	}
	notes := strings.TrimSpace(req.Notes)
	if notes == "" {
		notes = "Thank you for your business."
	}
	logoPath := a.engagementLetterLogoPath()
	logoWidth, logoHeight := 0, 0
	if logoBytes, openErr := a.secureReadFile(logoPath); openErr == nil {
		if cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(logoBytes)); cfgErr == nil {
			logoWidth, logoHeight = cfg.Width, cfg.Height
		} else {
			logoPath = ""
		}
	} else {
		logoPath = ""
	}
	total := rate
	if feeType == "Hourly Rate" {
		total = rate * hours
	}
	return invoiceData{
		CompanyName: companyName, ClientID: props.ClientID, ClientName: props.Name, ClientPath: props.Path,
		ClientAddress: props.Address, ClientPhone: props.Phone, ClientEmail: props.Email,
		PreparerName: preparerName, PreparerAddress: preparerAddress, PreparerPhone: phone, PreparerEmail: strings.TrimSpace(preparer.Email),
		Service: service, ServiceDesc: serviceDesc, TaxYear: taxYear, InvoiceNumber: invoiceNumber,
		InvoiceDate: invoiceDate.Format("January 2, 2006"), DueDate: dueDate.Format("January 2, 2006"),
		FeeType: feeType, Rate: rate, Hours: hours, Total: total, Description: description, Notes: notes,
		LogoPath: logoPath, LogoWidth: logoWidth, LogoHeight: logoHeight,
	}, nil
}

func invoiceParseNumber(value string) (float64, error) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "$", "")
	value = strings.ReplaceAll(value, ",", "")
	if value == "" {
		return 0, errors.New("empty number")
	}
	return strconv.ParseFloat(value, 64)
}

func invoiceMoneyText(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

func writeInvoicePDF(path string, data invoiceData) error {
	const pageWidth = 612.0
	const pageHeight = 792.0
	const left = 54.0
	const right = 54.0
	usable := pageWidth - left - right
	var stream strings.Builder
	y := pageHeight - 54.0
	if data.LogoPath != "" && data.LogoWidth > 0 && data.LogoHeight > 0 {
		logoW := 112.0
		logoH := logoW * float64(data.LogoHeight) / float64(data.LogoWidth)
		if logoH > 58.0 {
			logoH = 58.0
			logoW = logoH * float64(data.LogoWidth) / float64(data.LogoHeight)
		}
		stream.WriteString(fmt.Sprintf("q %.2f 0 0 %.2f %.2f %.2f cm /Im1 Do Q\n", logoW, logoH, left, y-logoH))
		y -= logoH + 8
	}
	write := func(text string, bold bool, size, x float64) {
		font := "F1"
		if bold {
			font = "F2"
		}
		stream.WriteString(fmt.Sprintf("BT /%s %.2f Tf 1 0 0 1 %.2f %.2f Tm (%s) Tj ET\n", font, size, x, y, engagementPDFEscape(text)))
	}
	write(data.CompanyName, true, 14, left)
	invoiceTitle := "INVOICE"
	write(invoiceTitle, true, 24, pageWidth-right-engagementApproxTextWidth(invoiceTitle, 24))
	y -= 18
	write(data.PreparerName, true, 9.5, left)
	y -= 13
	for _, line := range data.PreparerAddress {
		write(line, false, 9, left)
		y -= 12
	}
	for _, line := range []string{data.PreparerPhone, data.PreparerEmail} {
		if strings.TrimSpace(line) != "" {
			write(line, false, 9, left)
			y -= 12
		}
	}
	y -= 8
	stream.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n", left, y, pageWidth-right, y))
	y -= 22
	write("Invoice #: "+data.InvoiceNumber, true, 10, left)
	write("Invoice Date: "+data.InvoiceDate, false, 9.5, left+250)
	y -= 16
	write("Tax Year: "+data.TaxYear, false, 9.5, left)
	write("Due Date: "+data.DueDate, true, 9.5, left+250)
	y -= 28
	write("BILL TO", true, 9, left)
	y -= 15
	write(data.ClientName, true, 11, left)
	y -= 14
	for _, line := range []string{data.ClientAddress, data.ClientPhone, data.ClientEmail} {
		if strings.TrimSpace(line) != "" {
			for _, wrapped := range engagementWrapText(line, usable*0.72, 9.5) {
				write(wrapped, false, 9.5, left)
				y -= 13
			}
		}
	}
	y -= 18
	stream.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n", left, y, pageWidth-right, y))
	y -= 18
	write("DESCRIPTION", true, 9, left)
	write("QTY", true, 9, left+310)
	write("RATE", true, 9, left+365)
	write("AMOUNT", true, 9, left+435)
	y -= 14
	stream.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n", left, y, pageWidth-right, y))
	y -= 19
	descLines := engagementWrapText(data.Description, 290, 9.5)
	lineY := y
	for _, line := range descLines {
		write(line, false, 9.5, left)
		y -= 13
	}
	y = lineY
	qty := "1"
	if data.FeeType == "Hourly Rate" {
		qty = strconv.FormatFloat(data.Hours, 'f', 2, 64)
	}
	write(qty, false, 9.5, left+310)
	write(invoiceMoneyText(data.Rate), false, 9.5, left+365)
	write(invoiceMoneyText(data.Total), true, 9.5, left+435)
	if len(descLines) == 0 {
		y -= 13
	} else {
		y = lineY - float64(len(descLines))*13
	}
	y -= 8
	stream.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n", left, y, pageWidth-right, y))
	y -= 30
	label := "TOTAL DUE"
	write(label, true, 12, left+330)
	total := invoiceMoneyText(data.Total)
	write(total, true, 14, pageWidth-right-engagementApproxTextWidth(total, 14))
	y -= 40
	if data.Notes != "" {
		write("NOTE", true, 9, left)
		y -= 15
		for _, line := range engagementWrapText(data.Notes, usable, 9.5) {
			write(line, false, 9.5, left)
			y -= 13
		}
	}
	stream.WriteString(fmt.Sprintf("BT /F1 7.5 Tf 1 0 0 1 %.2f 28 Tm (Invoice %s) Tj ET\n", left, engagementPDFEscape(data.InvoiceNumber)))
	return writeSimplePDF(path, []string{stream.String()}, data.CompanyName, "Invoice "+data.InvoiceNumber+" - "+data.ClientName, data.LogoPath, data.LogoWidth, data.LogoHeight)
}

func writeInvoiceDOCX(path string, data invoiceData) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(file)
	closeWithError := func(inner error) error {
		zipErr := zw.Close()
		fileErr := file.Close()
		if inner != nil {
			return inner
		}
		if zipErr != nil {
			return zipErr
		}
		return fileErr
	}
	writeEntry := func(name, body string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, body)
		return err
	}
	var logoBytes []byte
	if data.LogoPath != "" && data.LogoWidth > 0 && data.LogoHeight > 0 {
		logoBytes, _ = os.ReadFile(data.LogoPath)
	}
	hasLogo := len(logoBytes) > 0
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/>`
	if hasLogo {
		contentTypes += `<Default Extension="jpg" ContentType="image/jpeg"/>`
	}
	contentTypes += `<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/></Types>`
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
	wordRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`
	if hasLogo {
		wordRels += `<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/firm-logo.jpg"/>`
	}
	wordRels += `</Relationships>`
	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr></w:style></w:styles>`
	core := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>%s</dc:title><dc:creator>%s</dc:creator><cp:lastModifiedBy>BabyFileCab</cp:lastModifiedBy><dcterms:created xsi:type="dcterms:W3CDTF">%s</dcterms:created></cp:coreProperties>`, xmlText("Invoice "+data.InvoiceNumber+" - "+data.ClientName), xmlText(data.CompanyName), time.Now().UTC().Format(time.RFC3339))
	app := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>BabyFileCab</Application></Properties>`
	document := invoiceDOCXDocument(data, hasLogo)
	for _, item := range []struct{ name, body string }{
		{"[Content_Types].xml", contentTypes}, {"_rels/.rels", rels}, {"word/document.xml", document},
		{"word/styles.xml", styles}, {"word/_rels/document.xml.rels", wordRels}, {"docProps/core.xml", core}, {"docProps/app.xml", app},
	} {
		if err := writeEntry(item.name, item.body); err != nil {
			return closeWithError(err)
		}
	}
	if hasLogo {
		w, err := zw.Create("word/media/firm-logo.jpg")
		if err != nil {
			return closeWithError(err)
		}
		if _, err := w.Write(logoBytes); err != nil {
			return closeWithError(err)
		}
	}
	return closeWithError(nil)
}

func invoiceDOCXDocument(data invoiceData, hasLogo bool) string {
	var body strings.Builder
	if hasLogo {
		body.WriteString(docxLogoParagraph(data.LogoWidth, data.LogoHeight))
	}
	body.WriteString(docxParagraph(data.CompanyName, true, "left", 28, 20))
	body.WriteString(docxParagraph(data.PreparerName, true, "left", 19, 0))
	for _, line := range data.PreparerAddress {
		body.WriteString(docxParagraph(line, false, "left", 18, 0))
	}
	if data.PreparerPhone != "" {
		body.WriteString(docxParagraph(data.PreparerPhone, false, "left", 18, 0))
	}
	if data.PreparerEmail != "" {
		body.WriteString(docxParagraph(data.PreparerEmail, false, "left", 18, 120))
	}
	body.WriteString(docxParagraph("INVOICE", true, "right", 38, 40))
	body.WriteString(docxParagraph("Invoice #: "+data.InvoiceNumber, true, "right", 20, 0))
	body.WriteString(docxParagraph("Invoice Date: "+data.InvoiceDate, false, "right", 19, 0))
	body.WriteString(docxParagraph("Due Date: "+data.DueDate, true, "right", 19, 160))
	body.WriteString(docxParagraph("BILL TO", true, "left", 18, 40))
	body.WriteString(docxParagraph(data.ClientName, true, "left", 22, 0))
	for _, line := range []string{data.ClientAddress, data.ClientPhone, data.ClientEmail} {
		if strings.TrimSpace(line) != "" {
			body.WriteString(docxParagraph(line, false, "left", 18, 0))
		}
	}
	body.WriteString(docxParagraph("Service", true, "left", 20, 30))
	body.WriteString(docxParagraph(data.Description, false, "left", 20, 0))
	body.WriteString(docxParagraph("Tax Year: "+data.TaxYear, false, "left", 19, 0))
	if data.FeeType == "Hourly Rate" {
		body.WriteString(docxParagraph(fmt.Sprintf("Hours: %.2f", data.Hours), false, "left", 19, 0))
		body.WriteString(docxParagraph("Hourly Rate: "+invoiceMoneyText(data.Rate), false, "left", 19, 80))
	} else {
		body.WriteString(docxParagraph("Flat Fee: "+invoiceMoneyText(data.Rate), false, "left", 19, 80))
	}
	body.WriteString(docxParagraph("TOTAL DUE: "+invoiceMoneyText(data.Total), true, "right", 28, 180))
	if data.Notes != "" {
		body.WriteString(docxParagraph("Note", true, "left", 19, 30))
		body.WriteString(docxParagraph(data.Notes, false, "left", 19, 0))
	}
	body.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="936" w:right="1123" w:bottom="936" w:left="1123" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr>`)
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:body>` + body.String() + `</w:body></w:document>`
}

func (a *App) buildEngagementLetterData(req EngagementLetterRequest) (engagementLetterData, error) {
	props, err := a.GetClientProperties(req.ClientPath)
	if err != nil {
		return engagementLetterData{}, err
	}
	service := strings.TrimSpace(req.Service)
	validService := false
	for _, item := range []string{"1040", "1041", "1065", "1120S", "1120", "Bookkeeping", "Tax Projection"} {
		if service == item {
			validService = true
			break
		}
	}
	if !validService {
		return engagementLetterData{}, errors.New("select a valid Engagement Letter service")
	}
	taxYear := strings.TrimSpace(req.TaxYear)
	yearNumber, yearErr := strconv.Atoi(taxYear)
	if yearErr != nil || yearNumber < 2018 || yearNumber > 2026 {
		return engagementLetterData{}, errors.New("select a tax year from 2018 through 2026")
	}
	feeType := strings.TrimSpace(req.FeeType)
	if feeType != "Flat Fee" && feeType != "Hourly Rate" {
		return engagementLetterData{}, errors.New("select Flat Fee or Hourly Rate")
	}
	feeValue := engagementFormatFee(feeType, req.FeeValue)
	if feeValue == "" {
		return engagementLetterData{}, errors.New("enter the flat fee or hourly rate")
	}

	users, err := a.ListCompanyUsers()
	if err != nil {
		return engagementLetterData{}, err
	}
	var preparer *UserProfile
	for i := range users {
		if strings.EqualFold(users[i].Username, strings.TrimSpace(req.PreparerUsername)) {
			copy := users[i]
			preparer = &copy
			break
		}
	}
	if preparer == nil {
		return engagementLetterData{}, errors.New("select a preparer from the current company")
	}

	// The engagement letter always uses the company account that is currently
	// signed into BabyFileCab. The preparer profile is used for the preparer's
	// personal/contact information, not to override the active firm name.
	companyName := ""
	a.authMu.RLock()
	if a.currentCompany != nil {
		companyName = strings.TrimSpace(a.currentCompany.Name)
	}
	a.authMu.RUnlock()
	if companyName == "" {
		companyName = strings.TrimSpace(preparer.CompanyName)
	}
	preparerName := strings.TrimSpace(preparer.FirstName + " " + preparer.LastName)
	if preparerName == "" {
		preparerName = preparer.Username
	}
	cityStateZIP := engagementCityStateZIP(*preparer)
	preparerAddress := make([]string, 0, 2)
	if strings.TrimSpace(preparer.StreetAddress) != "" {
		preparerAddress = append(preparerAddress, strings.TrimSpace(preparer.StreetAddress))
	}
	if cityStateZIP != "" {
		preparerAddress = append(preparerAddress, cityStateZIP)
	}
	phone := strings.TrimSpace(preparer.Telephone)
	if phone == "" {
		phone = strings.TrimSpace(preparer.Phone)
	}
	dateText := time.Now().Format("January 2, 2006")
	if raw := strings.TrimSpace(req.LetterDate); raw != "" {
		if parsed, parseErr := time.Parse("2006-01-02", raw); parseErr == nil {
			dateText = parsed.Format("January 2, 2006")
		} else {
			dateText = raw
		}
	}

	logoPath := a.engagementLetterLogoPath()
	logoWidth, logoHeight := 0, 0
	if logoBytes, openErr := a.secureReadFile(logoPath); openErr == nil {
		if cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(logoBytes)); cfgErr == nil {
			logoWidth, logoHeight = cfg.Width, cfg.Height
		} else {
			logoPath = ""
		}
	} else {
		logoPath = ""
	}

	return engagementLetterData{
		CompanyName: companyName, ClientID: props.ClientID, ClientName: props.Name, ClientPath: props.Path,
		ClientAddress: props.Address, ClientPhone: props.Phone, ClientEmail: props.Email,
		PreparerName: preparerName, PreparerAddress: preparerAddress, PreparerPhone: phone, PreparerEmail: strings.TrimSpace(preparer.Email),
		Service: service, ServiceDesc: engagementServiceDescription(service), TaxYear: taxYear,
		FeeType: feeType, FeeValue: feeValue, LetterDate: dateText,
		LogoPath: logoPath, LogoWidth: logoWidth, LogoHeight: logoHeight,
	}, nil
}

func engagementServiceDescription(service string) string {
	switch service {
	case "1040":
		return "Form 1040 U.S. Individual Income Tax Return"
	case "1041":
		return "Form 1041 U.S. Income Tax Return for Estates and Trusts"
	case "1065":
		return "Form 1065 U.S. Return of Partnership Income"
	case "1120S":
		return "Form 1120-S U.S. Income Tax Return for an S Corporation"
	case "1120":
		return "Form 1120 U.S. Corporation Income Tax Return"
	case "Bookkeeping":
		return "bookkeeping services"
	case "Tax Projection":
		return "tax projection services"
	default:
		return service
	}
}

func engagementCityStateZIP(user UserProfile) string {
	city := strings.TrimSpace(user.City)
	state := strings.TrimSpace(user.State)
	zipCode := strings.TrimSpace(user.ZIP)
	left := city
	if state != "" {
		if left != "" {
			left += ", " + state
		} else {
			left = state
		}
	}
	if zipCode != "" {
		if left != "" {
			left += " " + zipCode
		} else {
			left = zipCode
		}
	}
	return left
}

func engagementFormatFee(feeType, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "$") {
		value = "$" + value
	}
	lower := strings.ToLower(value)
	if feeType == "Hourly Rate" && !strings.Contains(lower, "/hr") && !strings.Contains(lower, "/hour") && !strings.Contains(lower, "per hour") {
		value += "/hr"
	}
	return value
}

func engagementSafeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Client"
	}
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' || r == ' ' {
			b.WriteRune(r)
		}
	}
	value = strings.Join(strings.Fields(b.String()), "_")
	if len(value) > 80 {
		value = value[:80]
	}
	if value == "" {
		return "Client"
	}
	return value
}

func engagementLetterSections(data engagementLetterData) (string, []engagementLetterSection, string) {
	company := data.CompanyName
	if company == "" {
		company = "the Firm"
	}
	client := data.ClientName
	if client == "" {
		client = "the Client"
	}
	intro := fmt.Sprintf("Thank you for engaging %s. This Engagement Letter confirms the terms, scope, and limitations of our engagement with %s for %s for tax year or period %s. Please review this letter carefully because it defines the responsibilities of both the Client and the Firm.", company, client, data.ServiceDesc, data.TaxYear)

	var scope string
	if engagementIsTaxReturnService(data.Service) {
		scope = fmt.Sprintf("%s will prepare the federal %s identified above from information, books, records, schedules, financial statements, and representations provided by %s. State, local, amended, informational, payroll, sales tax, foreign reporting, bookkeeping, notice response, audit, tax planning, tax projection, and other services are outside this engagement unless separately agreed to in writing.", company, data.ServiceDesc, client)
	} else if data.Service == "Bookkeeping" {
		scope = fmt.Sprintf("%s will provide bookkeeping services for %s using bank data, source documents, account information, classifications, instructions, and other records supplied or approved by the Client. The engagement may include recording transactions, reconciliations, and preparation of management reports or financial statements from those records. Unless separately agreed in writing, this engagement does not include an audit, review, compilation, tax return preparation, payroll, sales tax, or assurance service.", company, client)
	} else {
		scope = fmt.Sprintf("%s will prepare tax projections for %s using the income, deduction, withholding, estimated payment, transaction, and other assumptions supplied by the Client. A projection is an estimate based on information available at the time and is not a guarantee of the Client's final tax liability. Actual results may differ because of later transactions, incomplete information, changes in law, or other events.", company, client)
	}

	clientResponsibilities := fmt.Sprintf("%s is responsible for providing complete, accurate, and timely information and for maintaining the books, records, source documents, and supporting evidence required for the services and by applicable tax or regulatory authorities. The Client is responsible for the accuracy and completeness of all financial statements, trial balances, bookkeeping records, schedules, tax documents, classifications, estimates, and representations provided to the Firm, including information received from third parties. The Client must disclose all material facts and must review all returns, reports, projections, or other work product before relying on, signing, or filing it.", client)

	firmResponsibilities := fmt.Sprintf("%s will perform the agreed services using the information, financial statements, books, records, schedules, source documents, and representations supplied by %s or by third parties designated by the Client. Unless a separate written engagement expressly states otherwise, our services are not an audit, review, examination, agreed-upon procedures engagement, or other assurance service, and we will not independently verify or authenticate the information provided to us. We have no duty to discover fraud, embezzlement, misappropriation, illegal acts, errors, omissions, or weaknesses in internal control. Any financial statements, reports, or schedules used or produced from Client-provided accounting information remain the responsibility of Client management, including responsibility for their completeness, accuracy, accounting policies, classifications, estimates, and underlying records. Our work does not relieve the Client of any management, reporting, filing, payment, or recordkeeping responsibility.", company, client)

	reliance := fmt.Sprintf("%s may rely on information supplied by %s without independent verification unless we expressly agree otherwise in writing. Consistent with applicable professional standards, the Firm may make reasonable inquiries, request clarification, or ask for additional supporting documentation when information provided appears incorrect, incomplete, inconsistent, unusual, or otherwise requires clarification, and the Firm may suspend or decline to complete affected work until it is reasonably satisfied that sufficient relevant information has been provided. These inquiries do not constitute an audit, review, examination, or independent verification of the Client's records and do not shift responsibility for the accuracy and completeness of those records from the Client to the Firm. To the fullest extent permitted by applicable law, the Firm will not be responsible for taxes, interest, penalties, losses, damages, costs, missed deadlines, or other consequences arising from incomplete, inaccurate, misclassified, misleading, or late information; Client-prepared financial statements or bookkeeping records; data supplied by banks, payroll providers, investment firms, software systems, or other third parties; the Client's failure to disclose material facts; or the Client's failure to timely act on information or recommendations provided by the Firm. Nothing in this Engagement Letter is intended to limit any responsibility or liability that cannot lawfully be limited or waived.", company, client)

	sections := []engagementLetterSection{
		{Title: "Scope of Services", Body: scope},
		{Title: "Client Responsibilities", Body: clientResponsibilities},
		{Title: "Our Responsibilities and Reliance on Client-Provided Information", Body: firmResponsibilities},
		{Title: "Reliance, Limitations, and Risk Allocation", Body: reliance},
	}
	if engagementIsTaxReturnService(data.Service) {
		sections = append(sections,
			engagementLetterSection{Title: "Tax Positions and Professional Judgment", Body: fmt.Sprintf("%s may ask questions or request additional documentation when information appears incomplete, inconsistent, or unclear. The Firm may decline to take a tax position that it believes lacks adequate support or would cause the Firm to violate applicable law, regulations, or professional standards. The Client remains responsible for the positions reported on the return and for the taxes, interest, and penalties legally due.", company)},
			engagementLetterSection{Title: "Electronic Filing and Authorization", Body: fmt.Sprintf("If the return is eligible for electronic filing, %s will not transmit it until the Firm receives the required signed e-file authorization. Authorization to e-file does not transfer responsibility for the return from the Client to the Firm. The Client is responsible for reviewing the completed return and confirming that the information is complete and accurate before signing the authorization.", company)},
		)
	}
	sections = append(sections,
		engagementLetterSection{Title: "Deadlines, Extensions, and Client Action", Body: fmt.Sprintf("%s agrees to provide requested information sufficiently in advance of applicable deadlines. The Firm cannot guarantee timely completion when information is incomplete or received late. When applicable, an extension of time to file is not an extension of time to pay. Tax payments, estimated payments, elections, signatures, approvals, and other Client actions remain the Client's responsibility unless a separate written agreement specifically provides otherwise.", client)},
		engagementLetterSection{Title: "Fees and Additional Services", Body: fmt.Sprintf("The agreed fee arrangement for this engagement is %s: %s. This fee applies only to the Service identified in this Engagement Letter and is based on the scope, facts, records, and level of professional time reasonably anticipated when this engagement is accepted. If additional information, records, reconciliation, correction, research, analysis, consultation, or professional time is required beyond what was reasonably contemplated, or if the scope or complexity of the engagement materially changes, the Firm will discuss with the Client an additional flat fee or hourly rate before undertaking material out-of-scope work. Any such additional services and related compensation must be mutually agreed upon by the Client and the Firm, which agreement may be documented in writing, including by email, supplemental engagement terms, or other written authorization. Additional work may include amended returns, notices, audit or examination assistance, bookkeeping cleanup, tax research, tax planning, tax projections, or other services outside the stated scope. The Firm may pause work when requested information, authorization, or payment is outstanding, and resulting delays may require revised completion dates and additional fees.", data.FeeType, data.FeeValue)},
		engagementLetterSection{Title: "Records, Confidentiality, and Third-Party Systems", Body: fmt.Sprintf("%s will handle Client information in accordance with applicable professional and legal obligations. The Client should retain original records and supporting documentation. Electronic communication, portals, financial institutions, software providers, and other third-party systems may be used in performing the engagement; those systems are outside the Firm's direct control. The Firm is not responsible for the acts, omissions, outages, or security failures of third-party providers except to the extent responsibility cannot legally be disclaimed.", company)},
		engagementLetterSection{Title: "Termination", Body: fmt.Sprintf("Either %s or %s may terminate this engagement by written notice. The Client remains responsible for fees earned and costs incurred through the termination date and for all filing, payment, recordkeeping, and other deadlines after termination.", client, company)},
	)
	closing := fmt.Sprintf("If the terms above correctly describe your understanding of this engagement, please sign below. Your signature confirms that %s has read, understands, and accepts this Engagement Letter with %s.", client, company)
	return intro, sections, closing
}

func engagementIsTaxReturnService(service string) bool {
	switch service {
	case "1040", "1041", "1065", "1120S", "1120":
		return true
	default:
		return false
	}
}

func engagementPDFBlocks(data engagementLetterData) []engagementPDFBlock {
	intro, sections, closing := engagementLetterSections(data)
	blocks := []engagementPDFBlock{
		{Text: data.LetterDate, Size: 9.5, Align: "right", After: 8},
		{Text: data.CompanyName, Bold: true, Size: 14, After: 2},
		{Text: data.PreparerName, Bold: true, Size: 9.5, After: 1},
	}
	for _, line := range data.PreparerAddress {
		blocks = append(blocks, engagementPDFBlock{Text: line, Size: 9, After: 1})
	}
	if data.PreparerPhone != "" {
		blocks = append(blocks, engagementPDFBlock{Text: data.PreparerPhone, Size: 9, After: 1})
	}
	if data.PreparerEmail != "" {
		blocks = append(blocks, engagementPDFBlock{Text: data.PreparerEmail, Size: 9, After: 1})
	}
	blocks = append(blocks, engagementPDFBlock{Text: data.ClientName, Bold: true, Size: 9.5, Before: 12, After: 2})
	for _, line := range []string{data.ClientAddress, data.ClientPhone, data.ClientEmail} {
		if strings.TrimSpace(line) != "" {
			blocks = append(blocks, engagementPDFBlock{Text: line, Size: 9, After: 1})
		}
	}
	blocks = append(blocks,
		engagementPDFBlock{Text: "Engagement Letter", Bold: true, Size: 15, Align: "center", Before: 14, After: 10},
		engagementPDFBlock{Text: "Type of engagement: " + data.ServiceDesc, Bold: true, Size: 10, After: 2},
		engagementPDFBlock{Text: "Tax Year: " + data.TaxYear, Size: 10, After: 2},
		engagementPDFBlock{Text: data.FeeType + ": " + data.FeeValue, Size: 10, After: 10},
		engagementPDFBlock{Text: "Dear " + data.ClientName + ":", Size: 10, After: 8},
		engagementPDFBlock{Text: intro, Size: 10, After: 8},
	)
	for _, section := range sections {
		blocks = append(blocks,
			engagementPDFBlock{Text: section.Title, Bold: true, Size: 10.5, Before: 5, After: 3},
			engagementPDFBlock{Text: section.Body, Size: 9.6, After: 7},
		)
	}
	blocks = append(blocks,
		engagementPDFBlock{Text: closing, Size: 9.6, Before: 4, After: 18},
		engagementPDFBlock{Text: "Client acceptance:", Bold: true, Size: 10.5, After: 18},
		engagementPDFBlock{Text: data.ClientName + " signature: ____________________________________", Size: 10, After: 5},
		engagementPDFBlock{Text: "Date: ____________________", Size: 10, After: 18},
		engagementPDFBlock{Text: "Firm acknowledgement:", Bold: true, Size: 10.5, After: 18},
		engagementPDFBlock{Text: data.CompanyName + " / " + data.PreparerName + ": ____________________________________", Size: 10, After: 5},
		engagementPDFBlock{Text: "Date: ____________________", Size: 10},
	)
	return blocks
}

func writeEngagementLetterPDF(path string, data engagementLetterData) error {
	blocks := engagementPDFBlocks(data)
	const pageWidth = 612.0
	const pageHeight = 792.0
	const left = 54.0
	const right = 54.0
	const top = 54.0
	const bottom = 52.0
	usable := pageWidth - left - right

	pages := []*strings.Builder{&strings.Builder{}}
	y := pageHeight - top
	if data.LogoPath != "" && data.LogoWidth > 0 && data.LogoHeight > 0 {
		logoW := 120.0
		logoH := logoW * float64(data.LogoHeight) / float64(data.LogoWidth)
		if logoH > 62.0 {
			logoH = 62.0
			logoW = logoH * float64(data.LogoWidth) / float64(data.LogoHeight)
		}
		logoY := pageHeight - top - logoH
		pages[0].WriteString(fmt.Sprintf("q %.2f 0 0 %.2f %.2f %.2f cm /Im1 Do Q\n", logoW, logoH, left, logoY))
		y = logoY - 10
	}
	newPage := func() {
		pages = append(pages, &strings.Builder{})
		y = pageHeight - top
	}
	writeLine := func(text string, bold bool, size float64, align string) {
		leading := size * 1.34
		if y-leading < bottom {
			newPage()
		}
		x := left
		width := engagementApproxTextWidth(text, size)
		if align == "right" {
			x = pageWidth - right - width
		} else if align == "center" {
			x = left + (usable-width)/2
		}
		if x < left {
			x = left
		}
		font := "F1"
		if bold {
			font = "F2"
		}
		pages[len(pages)-1].WriteString(fmt.Sprintf("BT /%s %.2f Tf 1 0 0 1 %.2f %.2f Tm (%s) Tj ET\n", font, size, x, y, engagementPDFEscape(text)))
		y -= leading
	}

	for _, block := range blocks {
		y -= block.Before
		lines := engagementWrapText(block.Text, usable, block.Size)
		if len(lines) == 0 {
			lines = []string{""}
		}
		for _, line := range lines {
			writeLine(line, block.Bold, block.Size, block.Align)
		}
		y -= block.After
	}
	for i := range pages {
		pages[i].WriteString(fmt.Sprintf("BT /F1 7.5 Tf 1 0 0 1 %.2f 28 Tm (Engagement Letter) Tj ET\n", left))
		footer := fmt.Sprintf("Page %d", i+1)
		x := pageWidth - right - engagementApproxTextWidth(footer, 7.5)
		pages[i].WriteString(fmt.Sprintf("BT /F1 7.5 Tf 1 0 0 1 %.2f 28 Tm (%s) Tj ET\n", x, footer))
	}
	streams := make([]string, len(pages))
	for i := range pages {
		streams[i] = pages[i].String()
	}
	return writeSimplePDF(path, streams, data.CompanyName, "Engagement Letter - "+data.ClientName, data.LogoPath, data.LogoWidth, data.LogoHeight)
}

func engagementApproxTextWidth(text string, size float64) float64 {
	width := 0.0
	for _, r := range text {
		switch {
		case r == ' ':
			width += size * 0.28
		case strings.ContainsRune(".,:;!'|ilI()[]", r):
			width += size * 0.28
		case unicode.IsUpper(r):
			width += size * 0.60
		default:
			width += size * 0.50
		}
	}
	return width
}

func engagementWrapText(text string, maxWidth, size float64) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0)
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if engagementApproxTextWidth(candidate, size) <= maxWidth {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}

func engagementPDFEscape(value string) string {
	value = strings.NewReplacer(
		"’", "'", "‘", "'", "“", "\"", "”", "\"", "–", "-", "—", "-", "…", "...", "•", "-",
	).Replace(value)
	var b strings.Builder
	for _, r := range value {
		if r < 32 || r > 126 {
			b.WriteByte('?')
			continue
		}
		switch r {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeSimplePDF(path string, pageStreams []string, author, title, logoPath string, logoWidth, logoHeight int) error {
	if len(pageStreams) == 0 {
		pageStreams = []string{""}
	}
	var logoBytes []byte
	if logoPath != "" && logoWidth > 0 && logoHeight > 0 {
		if b, err := os.ReadFile(logoPath); err == nil {
			logoBytes = b
		}
	}

	pageCount := len(pageStreams)
	baseAfterPages := 5 + pageCount*2
	imageID := 0
	infoID := baseAfterPages
	if len(logoBytes) > 0 {
		imageID = baseAfterPages
		infoID = baseAfterPages + 1
	}
	objects := make([][]byte, infoID)
	objects[0] = []byte("<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		pageID := 5 + i*2
		kids = append(kids, fmt.Sprintf("%d 0 R", pageID))
	}
	objects[1] = []byte(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pageCount))
	objects[2] = []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	objects[3] = []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>")
	for i, stream := range pageStreams {
		pageID := 5 + i*2
		contentID := pageID + 1
		resources := "<< /Font << /F1 3 0 R /F2 4 0 R >>"
		if i == 0 && imageID > 0 {
			resources += fmt.Sprintf(" /XObject << /Im1 %d 0 R >>", imageID)
		}
		resources += " >>"
		objects[pageID-1] = []byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources %s /Contents %d 0 R >>", resources, contentID))
		objects[contentID-1] = []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream))
	}
	if imageID > 0 {
		var imageObj bytes.Buffer
		fmt.Fprintf(&imageObj, "<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n", logoWidth, logoHeight, len(logoBytes))
		imageObj.Write(logoBytes)
		imageObj.WriteString("\nendstream")
		objects[imageID-1] = imageObj.Bytes()
	}
	objects[infoID-1] = []byte(fmt.Sprintf("<< /Title (%s) /Author (%s) /Creator (BabyFileCab) >>", engagementPDFEscape(title), engagementPDFEscape(author)))

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", i+1)
		buf.Write(obj)
		buf.WriteString("\nendobj\n")
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R /Info %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, infoID, xref)
	return os.WriteFile(path, buf.Bytes(), 0600)
}

func writeEngagementLetterDOCX(path string, data engagementLetterData) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(file)
	closeWithError := func(base error) error {
		zipErr := zw.Close()
		fileErr := file.Close()
		if base != nil {
			return base
		}
		if zipErr != nil {
			return zipErr
		}
		return fileErr
	}
	writeEntry := func(name, body string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, body)
		return err
	}

	var logoBytes []byte
	if data.LogoPath != "" && data.LogoWidth > 0 && data.LogoHeight > 0 {
		logoBytes, _ = os.ReadFile(data.LogoPath)
	}
	hasLogo := len(logoBytes) > 0

	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>`
	if hasLogo {
		contentTypes += `<Default Extension="jpg" ContentType="image/jpeg"/>`
	}
	contentTypes += `<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
		`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>` +
		`<Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>` +
		`</Types>`
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>` +
		`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>` +
		`</Relationships>`
	wordRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`
	if hasLogo {
		wordRels += `<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/firm-logo.jpg"/>`
	}
	wordRels += `</Relationships>`
	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr></w:style>` +
		`</w:styles>`
	core := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>%s</dc:title><dc:creator>%s</dc:creator><cp:lastModifiedBy>BabyFileCab</cp:lastModifiedBy><dcterms:created xsi:type="dcterms:W3CDTF">%s</dcterms:created></cp:coreProperties>`, xmlText("Engagement Letter - "+data.ClientName), xmlText(data.CompanyName), time.Now().UTC().Format(time.RFC3339))
	app := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>BabyFileCab</Application></Properties>`
	document := engagementDOCXDocument(data, hasLogo)

	for _, item := range []struct{ name, body string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rels},
		{"word/document.xml", document},
		{"word/styles.xml", styles},
		{"word/_rels/document.xml.rels", wordRels},
		{"docProps/core.xml", core},
		{"docProps/app.xml", app},
	} {
		if err := writeEntry(item.name, item.body); err != nil {
			return closeWithError(err)
		}
	}
	if hasLogo {
		w, err := zw.Create("word/media/firm-logo.jpg")
		if err != nil {
			return closeWithError(err)
		}
		if _, err := w.Write(logoBytes); err != nil {
			return closeWithError(err)
		}
	}
	return closeWithError(nil)
}

func engagementDOCXDocument(data engagementLetterData, hasLogo bool) string {
	intro, sections, closing := engagementLetterSections(data)
	var body strings.Builder
	if hasLogo {
		body.WriteString(docxLogoParagraph(data.LogoWidth, data.LogoHeight))
	}
	body.WriteString(docxParagraph(data.LetterDate, false, "right", 19, 0))
	body.WriteString(docxParagraph(data.CompanyName, true, "left", 28, 20))
	body.WriteString(docxParagraph(data.PreparerName, true, "left", 19, 0))
	for _, line := range data.PreparerAddress {
		body.WriteString(docxParagraph(line, false, "left", 18, 0))
	}
	if data.PreparerPhone != "" {
		body.WriteString(docxParagraph(data.PreparerPhone, false, "left", 18, 0))
	}
	if data.PreparerEmail != "" {
		body.WriteString(docxParagraph(data.PreparerEmail, false, "left", 18, 120))
	}
	body.WriteString(docxParagraph(data.ClientName, true, "left", 19, 0))
	for _, line := range []string{data.ClientAddress, data.ClientPhone, data.ClientEmail} {
		if strings.TrimSpace(line) != "" {
			body.WriteString(docxParagraph(line, false, "left", 18, 0))
		}
	}
	body.WriteString(docxParagraph("Engagement Letter", true, "center", 30, 150))
	body.WriteString(docxParagraph("Type of engagement: "+data.ServiceDesc, true, "left", 20, 0))
	body.WriteString(docxParagraph("Tax Year: "+data.TaxYear, false, "left", 20, 0))
	body.WriteString(docxParagraph(data.FeeType+": "+data.FeeValue, false, "left", 20, 120))
	body.WriteString(docxParagraph("Dear "+data.ClientName+":", false, "left", 20, 100))
	body.WriteString(docxParagraph(intro, false, "left", 20, 120))
	for _, section := range sections {
		body.WriteString(docxParagraph(section.Title, true, "left", 21, 30))
		body.WriteString(docxParagraph(section.Body, false, "left", 20, 110))
	}
	body.WriteString(docxParagraph(closing, false, "left", 20, 260))
	body.WriteString(docxParagraph("Client acceptance:", true, "left", 21, 220))
	body.WriteString(docxParagraph(data.ClientName+" signature: ____________________________________", false, "left", 20, 50))
	body.WriteString(docxParagraph("Date: ____________________", false, "left", 20, 240))
	body.WriteString(docxParagraph("Firm acknowledgement:", true, "left", 21, 220))
	body.WriteString(docxParagraph(data.CompanyName+" / "+data.PreparerName+": ____________________________________", false, "left", 20, 50))
	body.WriteString(docxParagraph("Date: ____________________", false, "left", 20, 0))
	body.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="936" w:right="1123" w:bottom="936" w:left="1123" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr>`)
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:body>` + body.String() + `</w:body></w:document>`
}

func docxLogoParagraph(widthPx, heightPx int) string {
	if widthPx <= 0 || heightPx <= 0 {
		return ""
	}
	const emuPerInch = 914400.0
	maxWidth := 1.65 * emuPerInch
	maxHeight := 0.85 * emuPerInch
	cx := maxWidth
	cy := cx * float64(heightPx) / float64(widthPx)
	if cy > maxHeight {
		cy = maxHeight
		cx = cy * float64(widthPx) / float64(heightPx)
	}
	return fmt.Sprintf(`<w:p><w:pPr><w:jc w:val="left"/><w:spacing w:after="80"/></w:pPr><w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:effectExtent l="0" t="0" r="0" b="0"/><wp:docPr id="1" name="Firm Logo"/><wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="0" name="firm-logo.jpg"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip r:embed="rId2"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`, int64(cx), int64(cy), int64(cx), int64(cy))
}

func docxParagraph(text string, bold bool, align string, sizeHalfPoints int, afterTwips int) string {
	if sizeHalfPoints <= 0 {
		sizeHalfPoints = 20
	}
	var pPr strings.Builder
	if align != "" && align != "left" {
		pPr.WriteString(`<w:jc w:val="` + xmlText(align) + `"/>`)
	}
	if afterTwips > 0 {
		pPr.WriteString(fmt.Sprintf(`<w:spacing w:after="%d"/>`, afterTwips))
	}
	pPrXML := ""
	if pPr.Len() > 0 {
		pPrXML = "<w:pPr>" + pPr.String() + "</w:pPr>"
	}
	boldXML := ""
	if bold {
		boldXML = "<w:b/>"
	}
	return fmt.Sprintf(`<w:p>%s<w:r><w:rPr>%s<w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="%d"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, pPrXML, boldXML, sizeHalfPoints, xmlText(text))
}

func xmlText(value string) string {
	return html.EscapeString(value)
}

func (a *App) OpenDataFolder() error {
	if err := a.requireSignedIn(); err != nil {
		return err
	}
	if err := openPath(a.dataRoot); err != nil {
		return err
	}
	_ = a.appendAppAuditCurrent("", "", "Company data folder opened", "Local company workspace opened in the file manager")
	return nil
}

func (a *App) NodeType(path string) string { return a.pathType(path) }

func (a *App) nextClientID() int {
	entries, _ := os.ReadDir(a.dataRoot)
	max := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) >= 5 {
			if n, err := strconv.Atoi(name[:5]); err == nil && n > max {
				max = n
			}
		}
	}
	return max + 1
}

func (a *App) requireSignedIn() error {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	if a.currentUser == nil || time.Since(a.lastActivity) >= a.idleTimeoutLocked() {
		return errors.New("sign in to BabyFileCab first")
	}
	return nil
}

// TouchActivity is called only for real input events in the desktop webview.
// A stale session cannot be revived without entering the password again.
func (a *App) TouchActivity() bool {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	if a.currentUser == nil || time.Since(a.lastActivity) >= a.idleTimeoutLocked() {
		return false
	}
	a.lastActivity = time.Now()
	return true
}

func (a *App) SessionActive() bool {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	return a.currentUser != nil && time.Since(a.lastActivity) < a.idleTimeoutLocked()
}

func (a *App) safePath(path string) (string, error) {
	if err := a.requireSignedIn(); err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("no item selected")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	root, _ := filepath.Abs(a.dataRoot)
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", errors.New("path is outside BabyFileCab data folder")
	}
	relPath, _ := filepath.Rel(root, abs)
	for _, part := range splitRel(relPath) {
		if part == vaultDBName || part == vaultReadyName || part == accountsDBName || part == accountsReadyName || part == ".vault-keys" || part == ".migration-backups" || part == ".plaintext-preview" || part == ".users.json" || part == ".companies.json" || part == clientProfileFile || part == documentMetadataFile || part == firmCalendarFile {
			return "", errors.New("this is an internal BabyFileCab file")
		}
	}
	check := root
	for _, part := range splitRel(relPath) {
		check = filepath.Join(check, part)
		info, statErr := os.Lstat(check)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("symbolic links are not allowed inside the company vault")
		}
	}
	// When a migrated legacy company uses the storage root directly, keep the
	// application from traversing into other companies' workspace directories
	// or account metadata. (The operating-system user still owns local files.)
	storage, _ := filepath.Abs(a.storageRoot)
	if root == storage {
		companiesRoot := filepath.Join(storage, "companies")
		if abs == companiesRoot || strings.HasPrefix(abs, companiesRoot+string(os.PathSeparator)) ||
			abs == filepath.Join(storage, ".users.json") || abs == filepath.Join(storage, ".companies.json") {
			return "", errors.New("path belongs to BabyFileCab system or another company workspace")
		}
	}
	return filepath.Clean(abs), nil
}

func (a *App) requireType(path, want string) (string, error) {
	p, err := a.safePath(path)
	if err != nil {
		return "", err
	}
	if a.pathType(p) != want {
		return "", fmt.Errorf("select a %s first", want)
	}
	return p, nil
}

func (a *App) pathType(path string) string {
	p, err := a.safePath(path)
	if err != nil {
		return ""
	}
	info, err := os.Stat(p)
	if err != nil {
		return ""
	}
	if !info.IsDir() {
		return "file"
	}
	rel, err := filepath.Rel(a.dataRoot, p)
	if err != nil {
		return ""
	}
	parts := splitRel(rel)
	if len(parts) == 1 && isClientDirName(parts[0]) {
		return "client"
	}
	if len(parts) == 2 && parts[1] == permanentFolder {
		return "permanent"
	}
	if len(parts) == 2 && isTaxYear(parts[1]) {
		return "taxyear"
	}
	if len(parts) == 2 {
		return "section"
	}
	if len(parts) >= 3 {
		return "section"
	}
	return "folder"
}

func (a *App) uploadTarget(path string) (string, error) {
	p, err := a.safePath(path)
	if err != nil {
		return "", err
	}
	t := a.pathType(p)
	switch t {
	case "client":
		return filepath.Join(p, permanentFolder), nil
	case "permanent", "taxyear", "section":
		return p, nil
	case "file":
		return filepath.Dir(p), nil
	default:
		return "", errors.New("select a client, folder, tax year, section, or file")
	}
}

func (a *App) clientPathFor(path string) (string, error) {
	p, err := a.safePath(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(a.dataRoot, p)
	if err != nil {
		return "", err
	}
	parts := splitRel(rel)
	if len(parts) < 1 || !isClientDirName(parts[0]) {
		return "", errors.New("select an item under a client")
	}
	return filepath.Join(a.dataRoot, parts[0]), nil
}

func (a *App) readClientProfile(clientPath string) (ClientProfile, error) {
	clientPath, err := a.requireType(clientPath, "client")
	if err != nil {
		return ClientProfile{}, err
	}
	var profile ClientProfile
	err = a.metadataJSONGet("client-profile", a.documentMetadataKey(clientPath), &profile)
	if errors.Is(err, os.ErrNotExist) {
		return ClientProfile{}, nil
	}
	return profile, err
}

func (a *App) writeClientProfile(clientPath string, profile ClientProfile) error {
	clientPath, err := a.requireType(clientPath, "client")
	if err != nil {
		return err
	}
	return a.metadataJSONPut("client-profile", a.documentMetadataKey(clientPath), profile)
}

func (a *App) documentMetadataPath() string {
	return filepath.Join(a.dataRoot, documentMetadataFile)
}

func (a *App) documentMetadataKey(path string) string {
	path, err := a.safePath(path)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(a.dataRoot, path)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func (a *App) readDocumentMetadata() (map[string]string, error) {
	meta := map[string]string{}
	err := a.metadataJSONGet("document-metadata", "all", &meta)
	if errors.Is(err, os.ErrNotExist) {
		return meta, nil
	}
	return meta, err
}

func (a *App) writeDocumentMetadata(meta map[string]string) error {
	return a.metadataJSONPut("document-metadata", "all", meta)
}

func (a *App) moveDocumentMetadata(oldPath, newPath string) {
	meta, err := a.readDocumentMetadata()
	if err != nil {
		return
	}
	oldKey := a.documentMetadataKey(oldPath)
	newKey := a.documentMetadataKey(newPath)
	if value, ok := meta[oldKey]; ok {
		delete(meta, oldKey)
		meta[newKey] = value
		_ = a.writeDocumentMetadata(meta)
	}
}

func (a *App) removeDocumentMetadata(path string) {
	meta, err := a.readDocumentMetadata()
	if err != nil {
		return
	}
	key := a.documentMetadataKey(path)
	if _, ok := meta[key]; ok {
		delete(meta, key)
		_ = a.writeDocumentMetadata(meta)
	}
}

func (a *App) remapDocumentMetadataPrefix(oldPath, newPath string) {
	meta, err := a.readDocumentMetadata()
	if err != nil {
		return
	}
	oldKey := a.documentMetadataKey(oldPath)
	newKey := a.documentMetadataKey(newPath)
	changed := false
	for key, value := range meta {
		if key == oldKey || strings.HasPrefix(key, oldKey+"/") {
			suffix := strings.TrimPrefix(key, oldKey)
			delete(meta, key)
			meta[newKey+suffix] = value
			changed = true
		}
	}
	if changed {
		_ = a.writeDocumentMetadata(meta)
	}
}

func (a *App) removeDocumentMetadataPrefix(path string) {
	meta, err := a.readDocumentMetadata()
	if err != nil {
		return
	}
	prefix := a.documentMetadataKey(path)
	changed := false
	for key := range meta {
		if key == prefix || strings.HasPrefix(key, prefix+"/") {
			delete(meta, key)
			changed = true
		}
	}
	if changed {
		_ = a.writeDocumentMetadata(meta)
	}
}

func (a *App) richNotesPathFor(path string) (string, error) {
	cp, err := a.clientPathFor(path)
	if err != nil {
		return "", err
	}
	return filepath.Join(cp, richNotesFile), nil
}

func (a *App) appendAudit(path, action, details string) error {
	cp, err := a.clientPathFor(path)
	if err != nil {
		return err
	}
	clientID, clientName, _ := parseClientDirName(filepath.Base(cp))
	entry := AuditEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    strings.TrimSpace(action),
		Details:   strings.TrimSpace(details),
	}
	user, root := a.auditContext()

	a.auditMu.Lock()
	defer a.auditMu.Unlock()
	if err := a.appendAuditSQLite("client", clientID, entry); err != nil {
		return err
	}
	return a.appendAppAuditEntryUnlocked(root, AppAuditEntry{
		Timestamp:  entry.Timestamp,
		Username:   user.Username,
		UserName:   strings.TrimSpace(strings.Join([]string{user.FirstName, user.LastName}, " ")),
		Role:       user.Role,
		ClientID:   clientID,
		ClientName: clientName,
		Action:     entry.Action,
		Details:    entry.Details,
	})
}

// LogAppEvent records a company-wide event that is not tied to one client,
// such as opening a firm tool. It is intentionally exposed to the Wails UI.
func (a *App) LogAppEvent(action, details string) error {
	if err := a.requireSignedIn(); err != nil {
		return err
	}
	action = strings.TrimSpace(action)
	if action == "" {
		return errors.New("audit action is required")
	}
	return a.appendAppAuditCurrent("", "", action, details)
}

// GetAppAuditTrail returns newest-first company-wide BabyFileCab history.
func (a *App) GetAppAuditTrail() ([]AppAuditEntry, error) {
	if err := a.requireSignedIn(); err != nil {
		return nil, err
	}
	a.auditMu.Lock()
	defer a.auditMu.Unlock()
	entries := make([]AppAuditEntry, 0)
	if err := a.loadAuditSQLite("app", "", func(b []byte) error {
		var entry AppAuditEntry
		if err := json.Unmarshal(b, &entry); err != nil {
			return err
		}
		entries = append(entries, entry)
		return nil
	}); err != nil {
		return nil, err
	}
	sortAppAudit(entries)
	return entries, nil
}

func (a *App) auditContext() (UserProfile, string) {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	var user UserProfile
	if a.currentUser != nil {
		user = *a.currentUser
	}
	return user, a.dataRoot
}

func (a *App) appendAppAuditCurrent(clientID, clientName, action, details string) error {
	user, root := a.auditContext()
	if user.Username == "" || root == "" {
		return nil
	}
	return a.appendAppAuditFor(root, user, clientID, clientName, action, details)
}

func (a *App) appendAppAuditFor(root string, user UserProfile, clientID, clientName, action, details string) error {
	if strings.TrimSpace(root) == "" {
		return nil
	}
	entry := AppAuditEntry{
		Timestamp:  time.Now().Format(time.RFC3339),
		Username:   strings.TrimSpace(user.Username),
		UserName:   strings.TrimSpace(strings.Join([]string{user.FirstName, user.LastName}, " ")),
		Role:       strings.TrimSpace(user.Role),
		ClientID:   strings.TrimSpace(clientID),
		ClientName: strings.TrimSpace(clientName),
		Action:     strings.TrimSpace(action),
		Details:    strings.TrimSpace(details),
	}
	a.auditMu.Lock()
	defer a.auditMu.Unlock()
	return a.appendAppAuditEntryUnlocked(root, entry)
}

func (a *App) appendAppAuditEntryUnlocked(root string, entry AppAuditEntry) error {
	if entry.Action == "" {
		return nil
	}
	return a.appendAuditSQLite("app", "", entry)
}

func (a *App) clientPathByID(clientID string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return ""
	}
	entries, err := os.ReadDir(a.dataRoot)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id, _, ok := parseClientDirName(entry.Name())
		if ok && id == clientID {
			return filepath.Join(a.dataRoot, entry.Name())
		}
	}
	return ""
}

func displayAuditRole(role string) string {
	if strings.EqualFold(strings.TrimSpace(role), "administrator") {
		return "Administrator"
	}
	return "Staff"
}

func (a *App) notesPathFor(path string) (string, error) {
	p, err := a.safePath(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(a.dataRoot, p)
	if err != nil {
		return "", err
	}
	parts := splitRel(rel)
	if len(parts) < 1 || !isClientDirName(parts[0]) {
		return "", errors.New("select an item under a client")
	}
	np := filepath.Join(a.dataRoot, parts[0], permanentFolder, notesFile)
	if _, err := os.Stat(np); errors.Is(err, os.ErrNotExist) {
		_ = a.secureWriteFile(np, []byte(""))
	}
	return np, nil
}

func splitRel(rel string) []string {
	if rel == "." || rel == "" {
		return nil
	}
	return strings.FieldsFunc(rel, func(r rune) bool { return r == '/' || r == '\\' })
}

func parseClientDirName(s string) (clientID, displayName string, ok bool) {
	if len(s) < 9 || s[5:8] != " - " {
		return "", "", false
	}
	if _, err := strconv.Atoi(s[:5]); err != nil {
		return "", "", false
	}
	displayName = strings.TrimSpace(s[8:])
	if displayName == "" {
		return "", "", false
	}
	return s[:5], displayName, true
}

func isClientDirName(s string) bool {
	_, _, ok := parseClientDirName(s)
	return ok
}

func isTaxYear(s string) bool {
	if len(s) != 4 {
		return false
	}
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1900 && n <= 2200
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "\x00", "")
	s = r.Replace(s)
	return strings.TrimSpace(s)
}

func uniqueDestination(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	dir, ext := filepath.Dir(path), filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	for i := 2; ; i++ {
		p := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
	}
}

func readCSV(path string, maxRows int) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(bufio.NewReader(f))
	r.FieldsPerRecord = -1
	var rows [][]string
	for len(rows) < maxRows {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return rows, err
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

func pdfPageCount(path string) (int, error) {
	out, err := exec.Command("pdfinfo", path).Output()
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)
	m := re.FindStringSubmatch(string(out))
	if len(m) != 2 {
		return 0, errors.New("could not determine PDF page count")
	}
	return strconv.Atoi(m[1])
}

func docxText(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return "", err
		}
		defer r.Close()
		dec := xml.NewDecoder(r)
		var b strings.Builder
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			switch t := tok.(type) {
			case xml.CharData:
				b.WriteString(string(t))
			case xml.EndElement:
				if t.Name.Local == "p" {
					b.WriteString("\n")
				}
			}
		}
		return b.String(), nil
	}
	return "", errors.New("document.xml not found")
}

func readXLSX(path string, maxRows, maxCols int) ([][]string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	shared := []string{}
	if data, err := zipEntryBytes(zr.File, "xl/sharedStrings.xml"); err == nil {
		shared, _ = parseSharedStrings(data)
	}

	var sheetName string
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			if sheetName == "" || f.Name < sheetName {
				sheetName = f.Name
			}
		}
	}
	if sheetName == "" {
		return nil, errors.New("workbook has no worksheet")
	}
	data, err := zipEntryBytes(zr.File, sheetName)
	if err != nil {
		return nil, err
	}

	type xlsxRun struct {
		Text string `xml:"t"`
	}
	type xlsxInline struct {
		Text string    `xml:"t"`
		Runs []xlsxRun `xml:"r"`
	}
	type xlsxCell struct {
		Ref    string     `xml:"r,attr"`
		Type   string     `xml:"t,attr"`
		Value  string     `xml:"v"`
		Inline xlsxInline `xml:"is"`
	}
	type xlsxRow struct {
		Cells []xlsxCell `xml:"c"`
	}
	type worksheet struct {
		Rows []xlsxRow `xml:"sheetData>row"`
	}
	var ws worksheet
	if err := xml.Unmarshal(data, &ws); err != nil {
		return nil, err
	}

	var out [][]string
	for _, row := range ws.Rows {
		if len(out) >= maxRows {
			break
		}
		values := make([]string, 0)
		for _, cell := range row.Cells {
			col := xlsxColumnIndex(cell.Ref)
			if col < 0 || col >= maxCols {
				continue
			}
			for len(values) <= col {
				values = append(values, "")
			}
			val := cell.Value
			switch cell.Type {
			case "s":
				if idx, err := strconv.Atoi(strings.TrimSpace(cell.Value)); err == nil && idx >= 0 && idx < len(shared) {
					val = shared[idx]
				}
			case "inlineStr":
				var b strings.Builder
				b.WriteString(cell.Inline.Text)
				for _, run := range cell.Inline.Runs {
					b.WriteString(run.Text)
				}
				val = b.String()
			case "b":
				if strings.TrimSpace(cell.Value) == "1" {
					val = "TRUE"
				} else {
					val = "FALSE"
				}
			}
			values[col] = val
		}
		out = append(out, values)
	}
	return out, nil
}

func zipEntryBytes(files []*zip.File, name string) ([]byte, error) {
	for _, f := range files {
		if f.Name != name {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	}
	return nil, os.ErrNotExist
}

func parseSharedStrings(data []byte) ([]string, error) {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var out []string
	var current strings.Builder
	inSI := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "si" {
				inSI = true
				current.Reset()
			} else if inSI && t.Name.Local == "t" {
				var text string
				if err := dec.DecodeElement(&text, &t); err != nil {
					return nil, err
				}
				current.WriteString(text)
			}
		case xml.EndElement:
			if t.Name.Local == "si" && inSI {
				out = append(out, current.String())
				inSI = false
			}
		}
	}
	return out, nil
}

func xlsxColumnIndex(ref string) int {
	col := 0
	seen := false
	for _, r := range strings.ToUpper(ref) {
		if r < 'A' || r > 'Z' {
			break
		}
		seen = true
		col = col*26 + int(r-'A'+1)
	}
	if !seen {
		return -1
	}
	return col - 1
}

func emlText(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	msg, err := mail.ReadMessage(f)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, key := range []string{"From", "To", "Cc", "Date", "Subject"} {
		if value := strings.TrimSpace(msg.Header.Get(key)); value != "" {
			fmt.Fprintf(&b, "%s: %s\n", key, value)
		}
	}
	b.WriteString("\n")
	body, err := readEmailBody(msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"), msg.Body)
	if err != nil {
		return b.String(), nil
	}
	b.WriteString(body)
	return b.String(), nil
}

func readEmailBody(contentType, transferEncoding string, r io.Reader) (string, error) {
	mediaType, params, _ := mime.ParseMediaType(contentType)
	mediaType = strings.ToLower(mediaType)
	if strings.HasPrefix(mediaType, "multipart/") && params["boundary"] != "" {
		mr := multipart.NewReader(r, params["boundary"])
		var htmlFallback string
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			text, err := readEmailBody(part.Header.Get("Content-Type"), part.Header.Get("Content-Transfer-Encoding"), part)
			_ = part.Close()
			if err != nil || strings.TrimSpace(text) == "" {
				continue
			}
			pt, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			if strings.EqualFold(pt, "text/plain") {
				return text, nil
			}
			if strings.EqualFold(pt, "text/html") && htmlFallback == "" {
				htmlFallback = text
			}
		}
		return htmlFallback, nil
	}

	decoded := decodeEmailTransfer(r, transferEncoding)
	data, err := io.ReadAll(decoded)
	if err != nil {
		return "", err
	}
	text := string(data)
	if mediaType == "text/html" {
		text = stripHTML(text)
	}
	return text, nil
}

func decodeEmailTransfer(r io.Reader, encoding string) io.Reader {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, r)
	case "quoted-printable":
		return quotedprintable.NewReader(r)
	default:
		return r
	}
}

func stripHTML(s string) string {
	re := regexp.MustCompile(`(?s)<[^>]*>`)
	s = re.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

func openPath(path string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
