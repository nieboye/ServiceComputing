package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	flag "github.com/spf13/pflag"
)

type sp_args struct {
	start_page  int //开始页码 
	end_page    int //结束页码
	in_filename string // 文件名
	page_len    int // 每一页的大小
	page_type   int // 页类型
	print_dest  string // 打印目的地
}

const INBUFSIZ = 16 * 1024

var progname string

func main() {
	sa := sp_args{-1, -1, "", 3, 'l', ""} //创建结构体数据
	progname = os.Args[0]
	process_args(len(os.Args), os.Args, &sa)
	process_input(sa)
}

func process_args(ac int, av []string, psa *sp_args) {
	var (
		str1, str2 string
		arg        int
	)
	if ac < 3 {
		fmt.Fprintln(os.Stderr, progname, ": not enough arguments")
		usage()
		os.Exit(1)
	}
	// handle 1st arg - start page
	str1 = av[1]
	if str1[:2] != "-s" {
		fmt.Fprintln(os.Stderr, progname, ": 1st arg should be -sstart_page")
		usage()
		os.Exit(2)
	}
	i, error := strconv.Atoi(str1[2:])
	if error != nil || i < 1 {
		fmt.Fprintln(os.Stderr, progname, ": invalid start page ", str1[2:])
		usage()
		os.Exit(3)
	}
	psa.start_page = i
	// handle 2nd arg - end page
	str1 = av[2]
	if str1[:2] != "-e" {
		fmt.Fprintln(os.Stderr, progname, ": 2nd arg should be -eend_page")
		usage()
		os.Exit(4)
	}
	i, error = strconv.Atoi(str1[2:])
	if error != nil || i < 1 || i < psa.start_page {
		fmt.Fprintln(os.Stderr, progname, ": invalid end page ", str1[2:])
		usage()
		os.Exit(5)
	}
	psa.end_page = i
	// handle optional args
	arg = 3
	for arg < ac && av[arg][0] == '-' {
		str1 = av[arg]
		switch str1[1] {
		case 'l':
			str2 := str1[2:]
			i, error = strconv.Atoi(str2)
			if error != nil || i < 1 {
				fmt.Fprintln(os.Stderr, progname, ": invalid page length ", str2)
				usage()
				os.Exit(6)
			}
			psa.page_len = i
			arg++
		case 'f':
			if str1 != "-f" {
				fmt.Fprintln(os.Stderr, progname, ": option should be \"-f\"")
				usage()
				os.Exit(7)
			}
			psa.page_type = 'f'
			arg++
		case 'd':
			str2 = str1[2:]
			if len(str2) < 1 {
				fmt.Fprintln(os.Stderr, progname, ": -d option requires a printer destination")
				usage()
				os.Exit(8)
			}
			psa.print_dest = str2
			arg++
		default:
			fmt.Fprintln(os.Stderr, progname, ": unknown option ", str1)
			usage()
			os.Exit(9)
		}
	}
	if arg < ac {
		psa.in_filename = av[arg]
		f, e := os.Open(psa.in_filename)
		if e != nil {
			panic(e)
		}
		defer f.Close()
	}
	//
	if !(psa.start_page > 0) {
		os.Exit(88)
	}
	if !(psa.end_page > 0 && psa.end_page >= psa.start_page) {
		os.Exit(88)
	}
	if !(psa.page_len > 1) {
		os.Exit(88)
	}
	if !(psa.page_type == 'l' || psa.page_type == 'f') {
		os.Exit(88)
	}
}

func process_input(sa sp_args) {
	fin, fout := os.Stdin, os.Stdout
	if len(sa.in_filename) != 0 {
		f, e := os.Open(sa.in_filename)
		if e != nil {
			panic(e)
		}
		fin = f
		defer f.Close()
	}
	if len(sa.print_dest) != 0 {
		sf := bufio.NewWriter(os.Stdout)
		sf.Flush()
		cmd := exec.Command("lp", "-d"+sa.print_dest)
		_, err := cmd.Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, progname, ": could not open pipe to \"lp -d", sa.print_dest, "\"")
			os.Exit(13)
		}
	}
	rf := bufio.NewReader(fin)
	wf := bufio.NewWriter(fout)
	fe := false
	var line_ctr, page_ctr int
	if sa.page_type == 'l' {
		line_ctr, page_ctr = 0, 1
		for {
			crc, err := rf.ReadString('\n')
			if err == io.EOF {
				break
			} else if err != nil {
				fe = true
				break
			}
			line_ctr++
			if line_ctr > sa.page_len {
				page_ctr++
				line_ctr = 1
			}
			if page_ctr >= sa.start_page && page_ctr <= sa.end_page {
				wf.WriteString(crc)
			}
		}
	} else {
		page_ctr = 1
		for {
			c, _, err := rf.ReadRune()
			if err == io.EOF {
				break
			} else if err != nil {
				fe = true
				break
			}
			if c == '\f' {
				page_ctr++
			}
			if page_ctr >= sa.start_page && page_ctr <= sa.end_page {
				wf.WriteRune(c)
			}
		}
	}
	if page_ctr < sa.start_page {
		fmt.Fprintln(os.Stderr, progname, ": start_page (", sa.start_page, ") greater than total pages (", page_ctr, "), no output written")
	} else if page_ctr < sa.end_page {
		fmt.Fprintln(os.Stderr, progname, ": end_page (", sa.end_page, ") greater than total pages (", page_ctr, "), less output than expected")
	}
	if fe {
		fmt.Fprintln(os.Stderr, progname, ": system error occurred on input stream fin")
	} else {
		fin.Close()
		wf.Flush()
		if len(sa.print_dest) != 0 {
			fout.Close()
		}
		fmt.Fprintln(os.Stderr, progname, ": done")
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "\nUSAGE: ", progname, " -s start_page -e end_page [ -f | -l lines_per_page ] [ -d dest ] [ in_filename ]")
}
