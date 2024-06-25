package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func main() {
	switch os.Args[1] {
	case "run":
		{
			run()
		}
	case "child":
		{
			child()
		}
	default:
		{
			panic("command not recognised")
		}
	}
}

func run() {
	fmt.Printf("Running command %v as %d\n", os.Args[2:], os.Getpid())

	cmd := exec.Command("proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	must(cmd.Run())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func child() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cg()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(syscall.Sethostname([]byte("container")))
	must(syscall.Chroot("")) // copy of ubuntu-fs
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	must(syscall.Mount("thing", "mytemp", "tmpfs", 0, ""))

	must(cmd.Run())

	must(syscall.Unmount("proc", 0))
	must(syscall.Unmount("thing", 0))
}

func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "user"), 0755)
	must(os.WriteFile(filepath.Join(pids, "user/pids.max"), []byte("20"), 0700))
	// Removes the new cgroup in place after the container exits
	must(os.WriteFile(filepath.Join(pids, "user/notify_on_release"), []byte("1"), 0700))
	must(os.WriteFile(filepath.Join(pids, "user/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}
