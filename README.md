# clashx-convert

将vmesss订阅转换为clashx配置

## 使用

### 下载源码

```shell script
git clone https://github.com/lifei6671/clash-convert.git
```

### 编译

```shell script
//用于将静态文件打包到二进制文件中
go generate
//Linux上编译Linux版本
GOOS=linux GOARCH=amd64 go build -o clashx-linux-amd64

//windows上编译为openwrt版本
set GOOS=linux
set GOOS=mipsle
set GOMIPS=softfloat
go build -o clashx-linux-mips64le
```

### 运行

```shell script
./clashx-linux-amd64 run --addr=":10200"
```