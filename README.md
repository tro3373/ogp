# Go OpenGraph CLI

dependency: dyatlov/go-opengraph: Golang package for parsing OpenGraph data from HTML into regular structures https://github.com/dyatlov/go-opengraph

Example usage:

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

