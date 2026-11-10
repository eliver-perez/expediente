package folderpicker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var ticketPattern = regexp.MustCompile(`^/[a-f0-9]{64}$`)

func Endpoint(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "aidex-pick" || u.Host != "select" || u.User != nil || u.Fragment != "" || u.RawPath != "" || !ticketPattern.MatchString(u.Path) {
		return "", fmt.Errorf("solicitud de selección inválida")
	}
	q, err := url.ParseQuery(u.RawQuery)
	port, e := strconv.Atoi(q.Get("port"))
	if err != nil || e != nil || len(q) != 1 || len(q["port"]) != 1 || port < 1 || port > 65535 || strconv.Itoa(port) != q.Get("port") {
		return "", fmt.Errorf("puerto de selección inválido")
	}
	return "http://127.0.0.1:" + strconv.Itoa(port) + "/api/v1/folder-selections" + u.Path, nil
}
func Call(ctx context.Context, endpoint string, body any, result any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("solicitud local rechazada (HTTP %d)", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 16384)).Decode(result)
}
func Run(raw string) error {
	endpoint, err := Endpoint(raw)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var result map[string]any
	if err = Call(ctx, endpoint+"/claim", struct{}{}, &result); err != nil {
		return err
	}
	verifyCtx, cancelVerify := context.WithTimeout(ctx, 10*time.Second)
	err = verifyLocal(verifyCtx, endpoint)
	cancelVerify()
	path := ""
	if err == nil {
		path, err = Choose(ctx)
	}
	code := ""
	if err != nil {
		code = "unavailable"
	} else if path == "" {
		code = "cancelled"
	}
	// The picker deadline may have expired. Always deliver a bounded terminal result.
	finish, cancelFinish := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFinish()
	return Call(finish, endpoint+"/finish", map[string]string{"server_path": path, "code": code}, &result)
}

// VerifyLocalService binds a loopback URL to this machine's installed service,
// including when a remote browser reaches the server through a tunnel.
func VerifyLocalService(ctx context.Context, endpoint string) error {
	return verifyLocal(ctx, endpoint)
}
