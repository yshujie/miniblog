#!/bin/bash

# 变量
PROJECTNAME="blog"
PROJECTBASE="."
PROJECTBIN="$PROJECTBASE/bin"
PROJECTLOGS="$PROJECTBASE/log"
prog=$PROJECTNAME

# 获取当前目录
CURDIR=$(pwd)
cd $CURDIR

# 确保日志目录存在
mkdir -p $PROJECTLOGS

# 运行服务
start() {
    echo -e "Begin to compile the project ---$PROJECTNAME ......"
    # 编译 Go 项目
    echo -e "---- Starting build $PROJECTNAME ... "
    go build -o $PROJECTNAME main.go
    # 赋予权限
    chmod 777 "$CURDIR/$PROJECTNAME"
    echo "Compilation completed"
    echo "Starting $PROJECTNAME, please wait..."
    # 后台运行项目
    nohup ./$PROJECTNAME > $PROJECTLOGS/run.log 2>&1 &
    echo -e "ok"
}

# 暂停服务
stop() {
    echo -e $"Stopping the project ---$prog: "
    # 获取进程
    pid=$(ps -ef | grep $prog | grep -v grep | awk '{print $2}')
    if [ "$pid" ]; then
        echo -n $"Kill process pid: $pid "
        # 杀掉进程
        kill -9 $pid
        ret=0
        # 多次循环杀掉进程
        for ((i=1; i<=15; i++)); do
            sleep 1
            pid=$(ps -ef | grep $prog | grep -v grep | awk '{print $2}')
            if [ "$pid" ]; then
                kill -9 $pid
                ret=0
            else
                ret=1
                break
            fi
        done

        if [ "$ret" ]; then
            echo -e $"ok"
        else
            echo -e $"no"
        fi
    else
        echo -e $"No program process to stop"
    fi
}

# 重启服务
restart() {
    stop
    sleep 2
    start
}

# 判断第一个参数
case "$1" in
start)
    start
    ;;
stop)
    stop
    ;;
restart)
    restart
    ;;
*)
    echo $"Usage: $0 {start|stop|restart}"
    exit 2
    ;;
esac
