# Go OpenGraph CLI

## Configuration

### X/Twitter

X/Twitter post URLs are resolved via the [FixTweet](https://github.com/FixTweet/FxTwitter)
API. No authentication is required.

- Post text comes back with `t.co` links already expanded.
- Link previews (title / description / image) come from the API response.
- If the API is unreachable, the URL falls back to normal OGP extraction.

To use a self-hosted FixTweet instance, set the API base URL either way:

```bash
# Environment variable
export OGP_FXTWITTER_API="https://fxtwitter.example.com"

# Config file at ~/.ogp
echo 'ogp_fxtwitter_api: "https://fxtwitter.example.com"' > ~/.ogp
```

A value that is not an absolute `http(s)` URL is ignored with a warning, and
the public instance is used instead.

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

