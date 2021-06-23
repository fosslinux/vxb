package util

import (
    "os/exec"
    "golang.org/x/sys/unix"
    "fmt"
    str "strings"
)

const TMPFS_MAGIC = 0x01021994

// Mount a tmpfs on a directory
func MountTmpfs(directory string, size string) error {
    err := unix.Mount("tmpfs", directory, "tmpfs", 0, "size=" + size)
    if err != nil {
        return fmt.Errorf("Error %w mounting %s as a tmpfs", err, directory)
    }
    return nil
}

// Mount a zram on a directory
func MountZram(directory string, size string, compression string) error {
    var err error

    // Firstly, create the zram device
    cmd, err := Sudo("zramctl", "-s", size, "-a", compression, "-f")
    if err != nil {
        return fmt.Errorf("Error %w creating zram device", err)
    }
    devRaw, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("Error %w creating zram device", err)
    }
    dev := str.TrimSpace(string(devRaw[:]))

    // TODO: Set the memlimit?
    // How to get root perms to write to /sys/block/zramx/mem_limit ideally
    // without shelling out to cat?

    // Create an ext4 filesystem on it
    cmd, err = Sudo("mkfs.ext4", "-O", "^has_journal", dev)
    if err != nil {
        return fmt.Errorf("Error %w creating ext4 filesystem on %s", err, dev)
    }
    err = cmd.Run()
    if err != nil {
        return fmt.Errorf("Error %w creating ext4 filesystem on %s", err, dev)
    }

    // Finally, mount that
    // The mount options ensure ext4 uses as little memory as possible
    err = unix.Mount(dev, directory, "ext4", 0, "discard,noatime,nodiratime")
    if err != nil {
        return fmt.Errorf("Error %w mounting %s on %s", err, dev, directory)
    }

    return nil
}

// Get data about zrams
func zramData() (string, error) {
    cmd := exec.Command("zramctl")
    data, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("Error %w executing zramctl", err)
    }
    return string(data[:]), nil
}

// Check if a mount is a zram mount
func isZram(directory string) (bool, error) {
    zramData, err := zramData()
    if err != nil {
        return false, fmt.Errorf("%w as checking if %s is zram mount", err, directory)
    }
    return str.Contains(zramData, " " + directory), nil
}

// Get the corresponding device to a directory (for zram)
func zramDev(directory string) (string, error) {
    var err error

    sData, err := zramData()
    if err != nil {
        return "", err
    }
    data := str.Split(sData, "\n")

    for _, line := range data {
        if str.Contains(line, " " + directory) {
            // This is the one with the data
            return str.Split(line, " ")[0], nil
        }
    }

    // If we got here, we couldn't get the devicea
    return "", fmt.Errorf("Could not find corresponding device to %s", directory)
}

// Unmount a filesystem
func Unmount(directory string) error {
    var err error

    // Unmount
    err = unix.Unmount(directory, 0)
    if err != nil {
        return fmt.Errorf("Error %w unmounting directory %s", err, directory)
    }

    // Check if it is a zram
    isZram, err := isZram(directory)
    if err != nil {
        return err
    }
    // If so, then we need to also destroy the zram
    if isZram {
        dev, err := zramDev(directory)
        if err != nil {
            return fmt.Errorf("Error %w destroying zram %s", err, directory)
        }
        cmd, err := Sudo("zramctl", "--reset", dev)
        if err != nil {
            return fmt.Errorf("Error %w destroying zram %s", err, directory)
        }
        err = cmd.Run()
        if err != nil {
            return fmt.Errorf("Error %w destroying zram %s", err, directory)
        }
    }

    return nil
}
