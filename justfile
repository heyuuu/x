# https://just.systems

# 列出可用命令
default:
    just --list

# 执行 x 工具
x *args:
    go -C cmd/x run . {{ args }}

# 创建新包: just init <name>
init name:
    just x init {{ name }}

# 在指定包下执行命令: just run <name> go test ./...
run name *args:
    just x run {{ name }} {{ args }}

# 在所有包下执行命令: just run-all go mod tidy
run-all *args:
    just x run-all {{ args }}

# 给指定包打 patch 版本 tag: just tag <name> [--minor] [--force]
tag name *flags:
    just x tag {{ name }} {{ flags }}

# 给所有包打 patch 版本 tag: just tag-all [--minor] [--force]
tag-all *flags:
    just x tag-all {{ flags }}

########################################
# 以下是各包的 fmt/lint/fix/test 四件套
########################################

# fmt
fmt name:
    just run {{ name }} go fmt ./...

fmt-all:
    just run-all go fmt ./...

# lint
lint name:
    just run {{ name }} go vet ./...
    just run {{ name }} golangci-lint run ./...

lint-all:
    just run-all go vet ./...
    just run-all golangci-lint run ./...

# fix
fix name:
    just run {{ name }} golangci-lint run --fix ./...

fix-all:
    just run-all golangci-lint run --fix ./...

# test
test name:
    just run {{ name }} go test -race ./...

test-all:
    just run-all go test -race ./...
