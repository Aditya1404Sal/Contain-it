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
	rootfs := fmt.Sprintf("./rootfs/%s", image)

	if _, err := os.Stat(rootfs); os.IsNotExist(err) {
		fmt.Printf("Root filesystem for %s not found. Pulling the image...\n", image)
		pullImage(image)
	}

	fmt.Printf("Running command %v in a new container\n", call)

	cmd := exec.Command("/proc/self/exe", append([]string{"child", rootfs, call}, os.Args[4:]...)...)
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
	must(syscall.Mount("tmpfs", "/tmp", "tmpfs", 0, ""))

	must(cmd.Run())

	must(syscall.Unmount("proc", 0))
	must(syscall.Unmount("/tmp", 0))
}

func pullImage(image string) {
	rootfsDir := fmt.Sprintf("./rootfs/%s", image)

	if _, err := os.Stat(rootfsDir); !os.IsNotExist(err) {
		fmt.Printf("Root filesystem for %s already exists. Skipping creation.\n", image)
		return
	}

	fmt.Printf("Creating root filesystem for %s...\n", image)
	must(os.MkdirAll(rootfsDir, 0755))

	// Use debootstrap to create the root filesystem
	cmd := exec.Command("sudo", "debootstrap",
		"--arch=amd64",
		image,
		rootfsDir,
		"http://archive.ubuntu.com/ubuntu/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())

	fmt.Printf("Root filesystem for %s created at %s\n", image, rootfsDir)

	fmt.Printf("Image %s pulled and root filesystem created successfully.\n", image)
}

func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")

	// Check if the cgroup filesystem is mounted
	if _, err := os.Stat(cgroups); os.IsNotExist(err) {
		fmt.Println("cgroup filesystem is not mounted. Attempting to mount...")
		must(os.MkdirAll(cgroups, 0755))
		must(syscall.Mount("cgroup", cgroups, "cgroup", 0, ""))
	}

	// Check if the pids subsystem exists
	if _, err := os.Stat(pids); os.IsNotExist(err) {
		fmt.Println("pids cgroup subsystem not found. Creating it...")
		must(os.MkdirAll(pids, 0755))
		must(os.WriteFile(filepath.Join(pids, "cgroup.procs"), []byte{}, 0600))
		must(os.WriteFile(filepath.Join(pids, "pids.max"), []byte("max"), 0600))
	}

	// Create a new cgroup for our container
	containerCgroup := filepath.Join(pids, "container")
	must(os.Mkdir(containerCgroup, 0755))
	must(os.WriteFile(filepath.Join(containerCgroup, "pids.max"), []byte("20"), 0600))
	// Removes the new cgroup in place after the container exits
	must(os.WriteFile(filepath.Join(containerCgroup, "notify_on_release"), []byte("1"), 0600))
	must(os.WriteFile(filepath.Join(containerCgroup, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0600))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
