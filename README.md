[![pipeline status](https://api.travis-ci.org/bityuan/bityuan.svg?branch=master)](https://travis-ci.org/bityuan/bityuan/)
[![Go Report Card](https://goreportcard.com/badge/github.com/bityuan/bityuan)](https://goreportcard.com/report/github.com/bityuan/bityuan)

# AIS公有链系统

#### 编译

```
git clone https://github.com/xiaoshitou520/aischain.git $GOPATH/src/github.com/xiaoshitou520/aischain
cd $GOPATH/src/github.com/xiaoshitou520/aischain
go build -i -o ais
go build -i -o ais-cli github.com/xiaoshitou520/aischain/cli
```

#### 运行

拷贝编译好的ais, ais-cli, aisChain.toml这三个文件置于同一个文件夹下，执行：

```
./ais -f aisChain.toml
```


