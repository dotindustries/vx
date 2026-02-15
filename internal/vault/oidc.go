package vault

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// oidcCallbackResult holds the outcome of a single OIDC callback invocation.
type oidcCallbackResult struct {
	code  string
	state string
	err   error
}

// OIDCAuth performs an OIDC authentication flow against Vault. It opens a
// browser for the user to authenticate, waits for the callback, and exchanges
// the authorization code for a Vault token. The token is set on the client.
func OIDCAuth(client *Client, role string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting OIDC callback listener: %w", err)
	}
	defer listener.Close()

	callbackPort := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/oidc/callback", callbackPort)

	authURL, clientNonce, err := requestAuthURL(client, role, redirectURI)
	if err != nil {
		return err
	}

	if err := openBrowser(authURL); err != nil {
		return fmt.Errorf("opening browser for OIDC login: %w", err)
	}

	result, err := waitForCallback(listener)
	if err != nil {
		return err
	}

	token, err := exchangeOIDCCode(client, result.code, result.state, clientNonce)
	if err != nil {
		return err
	}

	client.SetToken(token)
	return nil
}

// requestAuthURL calls Vault's auth/oidc/auth_url endpoint to get the URL
// the user must visit to authenticate.
func requestAuthURL(client *Client, role string, redirectURI string) (string, string, error) {
	data := map[string]interface{}{
		"role":         role,
		"redirect_uri": redirectURI,
	}

	secret, err := client.inner.Logical().Write("auth/oidc/auth_url", data)
	if err != nil {
		return "", "", fmt.Errorf("requesting OIDC auth URL: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", "", fmt.Errorf("requesting OIDC auth URL: empty response")
	}

	authURL, ok := secret.Data["auth_url"].(string)
	if !ok || authURL == "" {
		return "", "", fmt.Errorf("requesting OIDC auth URL: missing auth_url in response")
	}

	clientNonce, _ := secret.Data["client_nonce"].(string)

	return authURL, clientNonce, nil
}

// waitForCallback starts an HTTP server on the given listener and waits for
// the OIDC provider to redirect back with an authorization code.
func waitForCallback(listener net.Listener) (*oidcCallbackResult, error) {
	resultCh := make(chan oidcCallbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oidc/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			errMsg := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = r.URL.Query().Get("error")
			}
			resultCh <- oidcCallbackResult{err: fmt.Errorf("OIDC callback error: %s", errMsg)}
			fmt.Fprint(w, "Authentication failed. You may close this tab.")
			return
		}

		resultCh <- oidcCallbackResult{code: code, state: state}
		fmt.Fprint(w, "Authentication successful. You may close this tab.")
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	select {
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}
		return &result, nil
	case <-time.After(2 * time.Minute):
		return nil, fmt.Errorf("OIDC authentication timed out after 2 minutes")
	}
}

// exchangeOIDCCode exchanges the authorization code and state for a Vault token.
func exchangeOIDCCode(client *Client, code string, state string, clientNonce string) (string, error) {
	data := map[string]interface{}{
		"code":         code,
		"state":        state,
		"client_nonce": clientNonce,
	}

	secret, err := client.inner.Logical().Write("auth/oidc/callback", data)
	if err != nil {
		return "", fmt.Errorf("exchanging OIDC code for token: %w", err)
	}

	if secret == nil || secret.Auth == nil {
		return "", fmt.Errorf("exchanging OIDC code for token: empty auth response")
	}

	return secret.Auth.ClientToken, nil
}

// openBrowser opens the given URL in the user's default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform %q for opening browser", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launching browser: %w", err)
	}

	return nil
}
