package main

import (
    "flag"
    "github.com/ngaut/log"
    "tendisplus/integrate_test/util"
    "strconv"
)

func testRestore(m1_ip string, m1_port int, s1_ip string, s1_port int,
    s2_ip string, s2_port int, m2_ip string, m2_port int, kvstorecount int) {
    m1 := util.RedisServer{}
    s1 := util.RedisServer{}
    s2 := util.RedisServer{}
    m2 := util.RedisServer{}
    pwd := getCurrentDirectory()
    log.Infof("current pwd:" + pwd)
    m1.Init(m1_ip, m1_port, pwd, "m1_")
    s1.Init(s1_ip, s1_port, pwd, "s1_")
    s2.Init(s2_ip, s2_port, pwd, "s2_")
    m2.Init(m2_ip, m2_port, pwd, "m2_")

    cfgArgs := make(map[string]string)
    cfgArgs["kvstorecount"] = strconv.Itoa(kvstorecount)
    cfgArgs["requirepass"] = "tendis+test"
    cfgArgs["masterauth"] = "tendis+test"

    cfgArgs["maxbinlogkeepnum"] = "10000"
    cfgArgs["minbinlogkeepsec"] = "60"
    if err := m1.Setup(false, &cfgArgs); err != nil {
        log.Fatalf("setup master1 failed:%v", err)
    }

    cfgArgs["maxbinlogkeepnum"] = "10000"
    cfgArgs["minbinlogkeepsec"] = "60"
    if err := s1.Setup(false, &cfgArgs); err != nil {
        cfgArgs["maxbinlogkeepnum"] = "1"
        cfgArgs["minbinlogkeepsec"] = "0"
        log.Fatalf("setup slave1 failed:%v", err)
    }

    cfgArgs["maxbinlogkeepnum"] = "1"
    cfgArgs["minbinlogkeepsec"] = "0"
    if err := s2.Setup(false, &cfgArgs); err != nil {
        cfgArgs["maxbinlogkeepnum"] = "1"
        cfgArgs["minbinlogkeepsec"] = "0"
        log.Fatalf("setup slave2 failed:%v", err)
    }

    cfgArgs["maxbinlogkeepnum"] = "10000"
    cfgArgs["minbinlogkeepsec"] = "3600"
    if err := m2.Setup(false, &cfgArgs); err != nil {
        log.Fatalf("setup master2 failed:%v", err)
    }

    slaveof(&m1, &s1)
    waitFullsync(&s1, kvstorecount)

    slaveof(&s1, &s2)
    waitFullsync(&s2, kvstorecount)

    addData(&m1, *num1, "aa")

    waitCatchup(&m1, &s1, kvstorecount)
    waitCatchup(&s1, &s2, kvstorecount)

    backup(&s2)
    restoreBackup(&m2)

    var channel chan int = make(chan int)
    go compareInCoroutine(&m1, &s1, channel)
    go compareInCoroutine(&m1, &s2, channel)
    go compareInCoroutine(&m1, &m2, channel)
    <- channel
    <- channel
    <- channel

    addData(&m1, *num2, "bb")
    addOnekeyEveryStore(&m1, kvstorecount)

    waitCatchup(&m1, &s1, kvstorecount)
    waitCatchup(&s1, &s2, kvstorecount)

    waitDumpBinlog(&s2, kvstorecount)
    flushBinlog(&s2)
    restoreBinlog(&s2, &m2, kvstorecount)
    addOnekeyEveryStore(&m2, kvstorecount)
    compare(&m1, &m2)

    shutdownServer(&m1, *shutdown, *clear);
    shutdownServer(&s1, *shutdown, *clear);
    shutdownServer(&s2, *shutdown, *clear);
    shutdownServer(&m2, *shutdown, *clear);
}

func main(){
    flag.Parse()
    //rand.Seed(time.Now().UTC().UnixNano())
    testRestore(*m1ip, *m1port, *s1ip, *s1port, *s2ip, *s2port, *m2ip, *m2port, *kvstorecount)
}