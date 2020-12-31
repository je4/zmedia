#export PATH=$PATH:/usr/local/go/bin
#export GOROOT=/usr/local/go
#export GOPATH=/root/go
#go env
cd /mnt/daten/go/dev/zmedia
cp /dev/null log/dlv.log
dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient --log-dest=/dev/null --log ./cmd/server/ -- -cfg configs/mediaserver.toml > log/stdout.log 2> log/stderr.log
# dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient --log-dest=./log/dlv.log --log ./cmd/server/ -- -cfg configs/mediaserver.toml > log/stdout.log 2> log/stderr.log
# dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient --log ./cmd/server/ -- -cfg configs/mediaserver.toml > log/stdout.log 2> log/stderr.log


