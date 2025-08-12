# Go OpenGraph CLI

## Configuration

### X/Twitter Authentication

For X/Twitter URLs, authentication is required. There are two methods:

#### Method 1: Using cookies.json (Recommended)

1. Export cookies from your browser as JSON:
   - Chrome: Use extension like "Cookie-Editor" or "Export Cookies"
   - Firefox: Use "Cookie Quick Manager" extension
   - Export cookies for `x.com` domain

2. Configure the path to your cookies.json file (in order of priority):
   ```bash
   # Option 1: Environment variable (highest priority)
   export X_COOKIE_JSON="/path/to/your/cookies.json"

   # Option 2: Create config file at ~/.ogp
   echo 'x_cookie_json: "/path/to/your/cookies.json"' > ~/.ogp
   ```

3. Expected cookies.json format:
   ```json
   [
     {
       "name": "auth_token",
       "value": "your_auth_token_value",
       "domain": ".x.com",
       "path": "/",
       "secure": true,
       "httpOnly": true,
       "sameSite": "None"
     }
   ]
   ```

#### Method 2: Using Environment Variables

```bash
export X_AUTH_TOKEN="your_auth_token"
export X_CSRF_TOKEN="your_csrf_token"
```

Note: If no authentication is configured, the application will display an error message with all available configuration options including the X_COOKIE_JSON environment variable.

## Example usage:

```sh
ogp https://github.com/spf13/cobra-cli
```

```sh
cat<<EOF |ogp
https://github.com/spf13/cobra-cli
https://ja.wikipedia.org/wiki/Go_(%E3%83%97%E3%83%AD%E3%82%B0%E3%83%A9%E3%83%9F%E3%83%B3%E3%82%B0%E8%A8%80%E8%AA%9E)
https://go.dev/
EOF
```

