// Package facephys 封装 FacePhys 视频检测 API 的完整流程：
// presign → OSS 直传 → 处理。
//
// 用法（将本文件复制到你的项目中，package 名按需调整）:
//
//	client := facephys.New("your-key-id", "your-secret-key")
//	result, err := client.ProcessVideo(context.Background(), "/path/to/video.mp4")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if c := result.Data.Cardiac; c != nil {
//	    fmt.Printf("Heart rate: %.1f BPM\n", c.HR)
//	    fmt.Printf("Signal quality: %.2f\n", c.SQI)
//	}
//
// 零外部依赖，仅使用标准库，要求 Go 1.21+。
package facephys

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultBaseURL 是 FacePhys API 的默认地址。
	DefaultBaseURL = "https://www.facephys.com"
	// DefaultCapability 是默认处理能力。
	DefaultCapability = "rppg"
	// DefaultTimeout 是默认 HTTP 超时时间。
	DefaultTimeout = 120 * time.Second
)

// Error 表示一次 FacePhys API 调用失败。StatusCode 在网络层错误时为 0。
type Error struct {
	Message    string
	StatusCode int
}

func (e *Error) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("facephys: %s (status %d)", e.Message, e.StatusCode)
	}
	return "facephys: " + e.Message
}

// Client 是 FacePhys API 客户端。使用 New 构造，可被并发复用。
type Client struct {
	keyID      string
	secretKey  string
	baseURL    string
	capability string
	httpClient *http.Client
}

// Option 用于配置 Client。
type Option func(*Client)

// WithBaseURL 自定义 API 基础地址（默认 DefaultBaseURL）。
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithCapability 自定义默认处理能力（默认 DefaultCapability）。
func WithCapability(capability string) Option {
	return func(c *Client) { c.capability = capability }
}

// WithTimeout 自定义 HTTP 超时时间。与 WithHTTPClient 同时使用时，以后设置的为准。
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.httpClient = &http.Client{Timeout: timeout} }
}

// WithHTTPClient 注入自定义 *http.Client（例如配置代理或自定义传输层）。
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// New 创建一个 FacePhys 客户端。
func New(keyID, secretKey string, opts ...Option) *Client {
	c := &Client{
		keyID:      keyID,
		secretKey:  secretKey,
		baseURL:    DefaultBaseURL,
		capability: DefaultCapability,
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ── HMAC-SHA256 签名 ──

func (c *Client) sign(timestamp string) string {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(timestamp))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (c *Client) authHeaders() map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	return map[string]string{
		"x-key-id":    c.keyID,
		"x-timestamp": timestamp,
		"x-signature": c.sign(timestamp),
	}
}

// ── 核心流程 ──

type presignResponse struct {
	UploadURL string `json:"uploadUrl"`
	ObjectKey string `json:"objectKey"`
}

// ProcessVideo 使用客户端默认 capability 执行完整流程：
// presign → OSS 直传 → 提交处理。
func (c *Client) ProcessVideo(ctx context.Context, videoPath string) (*Result, error) {
	return c.ProcessVideoWithCapability(ctx, videoPath, c.capability)
}

// ProcessVideoWithCapability 与 ProcessVideo 相同，但允许覆盖本次调用的 capability。
func (c *Client) ProcessVideoWithCapability(ctx context.Context, videoPath, capability string) (*Result, error) {
	info, err := os.Stat(videoPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, &Error{Message: "video path is a directory: " + videoPath}
	}

	// Step 1: 获取 presign URL
	presign, err := c.presign(ctx)
	if err != nil {
		return nil, err
	}
	uploadURL := normalizeUploadURL(presign.UploadURL)
	if uploadURL == "" || presign.ObjectKey == "" {
		return nil, &Error{Message: "presign response missing uploadUrl or objectKey"}
	}

	// Step 2: 直传 OSS
	if err := c.uploadFile(ctx, uploadURL, videoPath, info.Size()); err != nil {
		return nil, err
	}

	// Step 3: 提交处理（重新签名，上传可能耗时较长）
	return c.process(ctx, presign.ObjectKey, capability)
}

func (c *Client) presign(ctx context.Context) (*presignResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v2/video/upload/presign", nil)
	if err != nil {
		return nil, &Error{Message: "build presign request: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.authHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Message: "presign request failed: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, &Error{
			Message:    "presign failed: " + apiErrorMessage(body, resp.StatusCode),
			StatusCode: resp.StatusCode,
		}
	}

	var out presignResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, &Error{Message: "decode presign response: " + err.Error()}
	}
	return &out, nil
}

func (c *Client) uploadFile(ctx context.Context, uploadURL, videoPath string, size int64) error {
	f, err := os.Open(videoPath)
	if err != nil {
		return err
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, f)
	if err != nil {
		return &Error{Message: "build upload request: " + err.Error()}
	}
	req.Header.Set("Content-Type", "video/mp4")
	// 显式设置 ContentLength，避免 OSS 因 chunked 传输拒绝请求。
	req.ContentLength = size

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &Error{Message: "OSS upload request failed: " + err.Error()}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return &Error{
			Message:    fmt.Sprintf("OSS upload failed: %d", resp.StatusCode),
			StatusCode: resp.StatusCode,
		}
	}
	return nil
}

func (c *Client) process(ctx context.Context, objectKey, capability string) (*Result, error) {
	payload, err := json.Marshal(map[string]string{
		"objectKey":  objectKey,
		"capability": capability,
	})
	if err != nil {
		return nil, &Error{Message: "encode process request: " + err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v3/video/process", bytes.NewReader(payload))
	if err != nil {
		return nil, &Error{Message: "build process request: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.authHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Message: "process request failed: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, &Error{
			Message:    "process failed: " + apiErrorMessage(body, resp.StatusCode),
			StatusCode: resp.StatusCode,
		}
	}

	var result Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, &Error{Message: "decode process response: " + err.Error()}
	}
	return &result, nil
}

// ── 工具方法 ──

// normalizeUploadURL 还原 uploadUrl path 中的 %2F，避免 OSS 签名路径不一致导致 403。
// 服务端偶发会把 path 里的 "/" 编码成 "%2F"，签名却是按 "/" 计算的，
// 这里只对 query 之前的 path 段还原，query 串原样保留以免破坏合法编码字符。
func normalizeUploadURL(url string) string {
	q := strings.IndexByte(url, '?')
	if q < 0 {
		return replaceEncodedSlash(url)
	}
	return replaceEncodedSlash(url[:q]) + url[q:]
}

func replaceEncodedSlash(s string) string {
	s = strings.ReplaceAll(s, "%2F", "/")
	return strings.ReplaceAll(s, "%2f", "/")
}

// apiErrorMessage 尽量从响应体中提取 {"error": "..."} 字段，否则回退到状态码。
func apiErrorMessage(body []byte, statusCode int) string {
	var parsed struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		return parsed.Error
	}
	if trimmed := strings.TrimSpace(string(body)); trimmed != "" {
		if len(trimmed) > 200 {
			trimmed = trimmed[:200]
		}
		return trimmed
	}
	return "status " + strconv.Itoa(statusCode)
}

// ── 返回结果结构 ──

// Result 是 V3 检测接口返回的分组结果。
//
// 实际字段取决于 API Key 开通的字段集，因此每个模块都是指针类型，
// 缺失模块为 nil。Raw 保留完整原始响应，便于读取未建模的新增字段。
type Result struct {
	Data            Data    `json:"data"`
	VideoDuration   float64 `json:"video_duration"`
	Message         string  `json:"message"`
	PointsDeducted  int64   `json:"points_deducted"`
	RemainingPoints int64   `json:"remaining_points"`

	// Raw 是完整的原始 JSON 响应。
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON 在填充类型化字段的同时保留原始字节。
func (r *Result) UnmarshalJSON(b []byte) error {
	type alias Result
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*r = Result(a)
	r.Raw = append(json.RawMessage(nil), b...)
	return nil
}

// Data 是各检测模块的容器。
type Data struct {
	Cardiac    *Cardiac           `json:"cardiac,omitempty"`
	BP         *BloodPressure     `json:"bp,omitempty"`
	SpO2       *SpO2              `json:"spo2,omitempty"`
	Psych      *Psych             `json:"psych,omitempty"`
	Emotion    *Emotion           `json:"emotion,omitempty"`
	FaceAU     map[string]float64 `json:"face_au,omitempty"`
	Behavior   *Behavior          `json:"behavior,omitempty"`
	Appearance *Appearance        `json:"appearance,omitempty"`
	Liveness   *Liveness          `json:"liveness,omitempty"`
}

// Cardiac 心率与心率变异性。
type Cardiac struct {
	HR     float64   `json:"hr"`
	SQI    float64   `json:"sqi"`
	HRList []HRPoint `json:"hr_list,omitempty"`
	HRV    *HRV      `json:"hrv,omitempty"`
}

// HRPoint 逐秒心率序列中的一个点。
type HRPoint struct {
	HR float64 `json:"hr"`
	TS float64 `json:"ts"`
}

// HRV 心率变异性指标。
type HRV struct {
	SDNN          float64 `json:"sdnn"`
	RMSSD         float64 `json:"rmssd"`
	PNN50         float64 `json:"pnn50"`
	LF            float64 `json:"LF"`
	HF            float64 `json:"HF"`
	LFHF          float64 `json:"LF/HF"`
	BreathingRate float64 `json:"breathing_rate"`
}

// BloodPressure 估算血压。
type BloodPressure struct {
	SBP        float64 `json:"sbp"`
	DBP        float64 `json:"dbp"`
	Confidence float64 `json:"confidence"`
}

// SpO2 估算血氧饱和度。
type SpO2 struct {
	SpO2       float64 `json:"spo2"`
	Confidence float64 `json:"confidence"`
}

// Psych 心理与状态评分。
type Psych struct {
	Stress        float64 `json:"stress"`
	Relaxation    float64 `json:"relaxation"`
	Fatigue       float64 `json:"fatigue"`
	SleepQuality  float64 `json:"sleep_quality"`
	Concentration float64 `json:"concentration"`
}

// Emotion 表情情绪与深层情绪。
type Emotion struct {
	Surface map[string]float64 `json:"surface,omitempty"`
	Deep    *EmotionDeep       `json:"deep,omitempty"`
}

// EmotionDeep 深层情绪推断。
type EmotionDeep struct {
	Dominant   string  `json:"dominant"`
	Confidence float64 `json:"confidence"`
}

// Behavior 眼动、疲劳与行为指标。
type Behavior struct {
	BlinkRate     float64 `json:"blink_rate"`
	Perclos       float64 `json:"perclos"`
	GazeStability float64 `json:"gaze_stability"`
}

// Appearance 人脸外观属性。
type Appearance struct {
	Age      *ValueConfidence `json:"age,omitempty"`
	Gender   *LabelConfidence `json:"gender,omitempty"`
	SkinTone *SkinTone        `json:"skin_tone,omitempty"`
}

// ValueConfidence 带置信度的数值。
type ValueConfidence struct {
	Value      float64 `json:"value"`
	Confidence float64 `json:"confidence"`
}

// LabelConfidence 带置信度的标签。
type LabelConfidence struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

// SkinTone 肤色（Fitzpatrick 分型）。
type SkinTone struct {
	Fitzpatrick string  `json:"fitzpatrick"`
	Confidence  float64 `json:"confidence"`
}

// Liveness 活体置信度与防伪信号。
type Liveness struct {
	IsLive        bool            `json:"is_live"`
	LivenessScore float64         `json:"liveness_score"`
	Signals       json.RawMessage `json:"signals,omitempty"`
}
