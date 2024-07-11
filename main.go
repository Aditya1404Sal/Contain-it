package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"

	"github.com/codeclysm/extract"
)

func main() {
	if len(os.Args) < 2 {
		panic("No command provided")
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 4 {
			panic("Usage: run <image> <command>")
		}
		image := os.Args[2]
		command := os.Args[3]
		run(image, command)
	case "child":
		child()
	case "pull":
		if len(os.Args) < 3 {
			panic("Usage: pull <image>")
		}
		image := os.Args[2]
		pullImage(image)
	default:
		panic(fmt.Sprintf("Command not recognized: %s", os.Args[1]))
	}
}

func run(image, call string) {
	tar := fmt.Sprintf("./assets/%s.tar.gz", image)

	if _, err := os.Stat(tar); errors.Is(err, os.ErrNotExist) {
		panic(err)
	}

	dir := createTempDir(tar)
	defer os.RemoveAll(dir)
	must(unTar(tar, dir))

	fmt.Printf("Running command %v in a new container\n", call)

	cmd := exec.Command("/proc/self/exe", append([]string{"child", dir, call}, os.Args[4:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}
	must(cmd.Run())
}

func child() {
	if len(os.Args) < 4 {
		panic("Not enough arguments for child process")
	}

	root := os.Args[2]
	call := os.Args[3]

	fmt.Printf("Running %v in new namespace\n", call)

	cg()

	cmd := exec.Command(call, os.Args[4:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(syscall.Sethostname([]byte("container")))
	must(syscall.Chroot(root))
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	must(syscall.Mount("thing", "mytemp", "tmpfs", 0, ""))

	must(cmd.Run())

	must(syscall.Unmount("proc", 0))
	must(syscall.Unmount("thing", 0))
}

func pullImage(image string) {
	cmd := exec.Command("./pull", image)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())
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

func createTempDir(name string) string {
	var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

	prefix := nonAlphanumericRegex.ReplaceAllString(name, "_")
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func unTar(source string, dst string) error {
	r, err := os.Open(source)
	if err != nil {
		return err
	}
	defer r.Close()

	ctx := context.Background()
	return extract.Archive(ctx, r, dst, nil)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
