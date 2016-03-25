package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"github.com/axgle/mahonia"
	"github.com/daviddengcn/go-colortext"
	"github.com/xcltapestry/xclpkg/clcolor"
)

//LogBlock 日志块 
type LogBlock struct {
	start int
	end   int
	index int
}

func cprint(c ct.Color, s string) {
	ct.Foreground(c, true)
	fmt.Println(s)
	ct.ResetColor()
}

func cprintf(c ct.Color, format string, args ...interface{}) {
	ct.Foreground(c, true)
	fmt.Printf(format, args...)
	ct.ResetColor()
}

type resultEntity struct {
	group   string
	content string
	time    string
	level   string
}

//GetPaths 获取所有日志地址
func GetPaths(p string, u bool) []string {
	var allLogs []string
	if ok, _, _ := isDir(p); ok {
		if u {
			allLogs = unzipAll(p)
		}
		allLogs = getFilelist(p, ".log")

	} else {
		if isLog(p) {
			allLogs = append(allLogs, p)
		} else {
			f, e := unzip(p)
			if e != nil {
				fmt.Println(e)
			} else {
				allLogs = append(allLogs, f)

			}
		}
	}
	return allLogs
}

//StartRead 开始读取日志
func StartRead(logfiles []string) []string {
	if len(logfiles) == 0 {
		return make([]string, 0)
	}
	chans := make(chan *resultEntity, 1000000)
	for _, v := range logfiles {
		go readLogger(v, chans)
	}
	return mergeLogger(chans, len(logfiles))
}

//QueryBlocks 根据搜索条件查找日志块
func QueryBlocks(search string, contents []string) []*LogBlock {
	var (
		indexs []int
		keys   []string
	)
	blocks := make(map[string]*LogBlock, 0)
	sortBlock := []*LogBlock{}
	for i, v := range contents {
		if strings.Index(v, search) > -1 {
			indexs = append(indexs, i)
		}
	}
	for _, v := range indexs {
		s, e := getBlockIndex(v, contents)
		_, _, stime, _ := getGroupName(contents[s])
		pkey := fmt.Sprintf("%s-%d-%d", stime, s, e)
		if _, ok := blocks[pkey]; !ok {
			blocks[pkey] = &LogBlock{start: s, end: e, index: v}
			keys = append(keys, pkey)
		}
	}
	sort.Sort(sort.StringSlice(keys))
	for i := len(keys) - 1; i >= 0; i-- {
		sortBlock = append(sortBlock, blocks[keys[i]])
	}
	return sortBlock
}

//GetBlockLogger 获取日志块
func GetBlockLogger(block *LogBlock, contents []string, search string) []string {
	var blockLogger []string
	for i := block.start; i <= block.end; i++ {
		blockLogger = append(blockLogger, strings.Replace(contents[i], search, clcolor.Green(search), 10))
	}
	return blockLogger
}



//------------------------------------------------------------------------------------
func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
func isDir(path string) (bool, string, error) {
	s, err := os.Stat(path)
	if err != nil {
		return false, "", err
	}
	return s.IsDir(), s.Name(), nil
}
func isGZ(name string) bool {
	return strings.HasSuffix(name, ".gz")
}
func isLog(name string) bool {
	return strings.HasSuffix(name, ".log")
}
func unzip(path string) (string, error) {
	p := path + ".log"
	if isExist(p) {
		return p, nil
	}
	fi, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fi.Close()

	fz, errx := gzip.NewReader(fi)
	if errx != nil {
		return "", errx
	}
	defer fz.Close()

	s, errr := ioutil.ReadAll(fz)
	if errr != nil {
		return "", errr
	}
	ioutil.WriteFile(p, s, 0666)
	return p, nil
}

func getFilelist(path string, p string) []string {
	var (
		strRet []string
		index  int
	)
	err := filepath.Walk(path, func(np string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if strings.HasSuffix(np, p) {
			strRet = append(strRet, np)
			index++
		}
		return nil
	})
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
	}
	sort.Sort(sort.StringSlice(strRet))
	return strRet
}

func unzipAll(p string) []string {
	var (
		all   []string
		index int
	)
	files := getFilelist(p, ".gz")
	logChan := make(chan string, len(files))
	if len(files) == 0 {
		return all
	}
	for _, v := range files {
		go func(fpath string) {
			f, e := unzip(fpath)
			if e != nil {
				fmt.Println(e)
			}
			logChan <- f
		}(v)
	}

	timePiker := time.NewTicker(time.Second * 2)
loop:
	for {
		select {
		case f := <-logChan:
			{
				index++
				if f != "" {
					all = append(all, f)
				}
				if index == len(files) {
					break loop
				}
			}
		case <-timePiker.C:
			{
				if index < len(files) {
					fmt.Printf("已解压:%d,剩余:%d\r\n", index, len(files)-index)
				}
			}

		}
	}
	return all
}

func getGroupName(l string) (line string, group string, time string, level string) {
	decoder := mahonia.NewDecoder("gbk")
	line = decoder.ConvertString(l)
	rs := []rune(line)
	group = string(rs[18:26])
	time = strings.Replace(string(rs[1:13]), "\n", "", 1)
	level = string(rs[15:16])
	return
}

//ReadLogger 从文件中读取日志
func readLogger(logfile string, resultChan chan<- *resultEntity) {
	s, _ := os.Stat(logfile)
	name := fmt.Sprintf("%d", s.ModTime().Unix())
	f, err := os.Open(logfile)
	if err == io.EOF {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	br := bufio.NewReader(f)
	for {
		l, err := br.ReadString('\n')
		if err != nil {
			break
		}
		line, group, time, level := getGroupName(l)
		entity := &resultEntity{content: line, group: group + name, time: time, level: level}
		resultChan <- entity
	}
	resultChan <- nil

}
func mergeLogger(resultChan chan *resultEntity, total int) []string {
	var (
		totalRecords []string
		keys         []string
		count        int
		chanCount    int
	)
	data := make(map[string][]string)
	timePiker := time.NewTicker(time.Second * 2)
loop:
	for {
		select {
		case entity := <-resultChan:
			{
				count++
				if entity == nil {
					chanCount++
					if chanCount == total {
						break loop
					}
					continue
				}

				if _, ok := data[entity.group]; !ok {
					data[entity.group] = make([]string, 0)
					keys = append(keys, entity.group)
				}
				data[entity.group] = append(data[entity.group], entity.content)
			}
		case <-timePiker.C:
			{
				fmt.Printf("已读取:%d行\r\n", count)
			}
		}
	}
	sort.Sort(sort.StringSlice(keys))
	for i := len(keys) - 1; i >= 0; i-- {
		for _, l := range data[keys[i]] {
			totalRecords = append(totalRecords, l)
		}
	}
	return totalRecords
}

func getBlockIndex(index int, contents []string) (start int, end int) {
	_, cgroup, _, _ := getGroupName(contents[index])
	start = index
	end = index
	for i := index - 1; i >= 0; i-- {
		_, group, _, _ := getGroupName(contents[i])
		if !strings.EqualFold(group, cgroup) {
			break
		}
		start = i
		if strings.Contains(contents[i], "-----") {
			break
		}
	}

	for i := index; i < len(contents); i++ {
		_, group, _, _ := getGroupName(contents[i])
		if !strings.EqualFold(group, cgroup) {
			break
		}
		if strings.Contains(contents[i], "-----") {
			break
		}
		end = i
		if strings.Contains(contents[i], "执行完成:") {
			break
		}
	}
	return
}
