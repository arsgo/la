package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"github.com/daviddengcn/go-colortext"
)

func main() {
    
	var (
		inputPath string
		unzip     bool
		find      string
	)
	flag.StringVar(&inputPath, "i", "", "输入文件或目录")
	flag.BoolVar(&unzip, "u", false, "是否解压目录下所有.gz文件")
	flag.StringVar(&find, "f", "", "匹配内容")
	flag.Parse()


	if len(inputPath) == 0 || len(find) == 0 {
		fmt.Println(`日志分析工具,读取日志文件或日志目录,并解析为日志块,可根据关健字搜索日志块信息`)
		flag.Usage()
		return
	}

	logs := GetPaths(inputPath, unzip)

	lines := StartRead(logs)
	fmt.Printf("共读取 %d个日志文件,总行数:%d\r\n", len(logs), len(lines))

	blocks := QueryBlocks(find, lines)
	fmt.Printf("根据关键字:'%s',共找到 %d 个日志块\r\n", find, len(blocks))

	if len(blocks) == 0 {
		return
	}

	cprint(ct.Red, "输入[回车]查看日志块,[q]退出")
	reader := bufio.NewReader(os.Stdin)
	b, _, _ := reader.ReadLine()
	if strings.EqualFold(string(b), "q") {
		return
	}

	for i := 0; i < len(blocks); i++ {
		v := blocks[i]
		block := GetBlockLogger(v, lines, find)

		for i, v := range block {
			if i != len(blocks)-1 {
				fmt.Println(v)
			}
		}
		if i < len(blocks)-1 {
			cprintf(ct.Red, "剩余:%d 个日志块,输入[回车]查看下一个,[q]退出\r\n", len(blocks)-i-1)
			b, _, _ := reader.ReadLine()
			if strings.EqualFold(string(b), "q") {
				break
			}
			fmt.Println()
		}
	}
}
