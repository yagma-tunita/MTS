// Package fileutil 提供文件系统操作工具函数。
//
// 包含的功能：
//   - 原子写文件（防崩溃导致文件损坏）
//   - 目录创建和存在性检查
//   - 文件/目录复制
//   - 跨平台文件锁（基于 flock）
//
// 设计原则：所有函数尽可能提供清晰的语义和错误信息。
// 例如 WriteFile 使用原子写入模式，确保写入过程中程序崩溃
// 不会留下不完整的文件。
package fileutil

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// EnsureDir 确保指定路径的目录存在，如果不存在则递归创建。
//
// 参数：
//   - path: 要确保存在的目录路径。可以是多级路径（如 "a/b/c"）。
//   - perm: 创建目录时使用的权限位，在 Windows 下部分权限位会被忽略。
//
// 内部调用 os.MkdirAll，类似于 Unix 的 mkdir -p。
// 如果 path 已经存在且是一个目录，返回 nil。
// 如果 path 存在但不是一个目录（比如是一个文件），返回错误。
//
// 使用场景：在写入文件前确保其父目录存在。
func EnsureDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Exists 检查文件或目录是否存在。
//
// 返回值：
//   - bool: true 表示文件/目录存在，false 表示不存在。
//   - error: 当权限不足或其他 I/O 错误导致无法判断时返回错误。
//     如果是"不存在"导致的错误（os.IsNotExist），返回 (false, nil)。
//
// 使用示例：
//
//	if ok, _ := fileutil.Exists("config.yaml"); !ok {
//	    fmt.Println("config file not found")
//	}
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// WriteFile 以原子方式写入文件。
//
// 什么是原子写入：
//   1. 先写入一个 .tmp 临时文件。
//   2. 写入成功后通过 os.Rename 重命名为目标文件名。
//   Rename 在同一个文件系统内是原子操作（POSIX 保证）。
//
// 为什么要原子写入：
//   如果直接写入目标文件（os.WriteFile），写入过程中程序崩溃
//   或断电，目标文件中可能只写入了部分数据，成为一个损坏的
//   文件。原子写入确保要么全部写入（.tmp 完全写好后改名），
//   要么全部不写入（.tmp 留在原地，目标文件不变）。
//
// 参数：
//   - filename: 目标文件路径。如果父目录不存在，会自动创建。
//   - data: 要写入的字节数据。
//   - perm: 文件权限位（Unix）。
//
// 注意：.tmp 文件与目标文件在同一个目录中。
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := EnsureDir(dir, 0755); err != nil {
		return err
	}
	tmp := filename + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, filename)
}

// CopyFile 复制文件从 src 到 dst。
//
// 实现方式：
//   1. 打开源文件（os.Open）。
//   2. 创建目标文件（os.Create）。
//   3. 使用 io.Copy 从源文件读取并写入目标文件。
//   4. 调用 Sync 确保数据刷入磁盘。
//
// Sync 的作用：io.Copy 可能因为操作系统缓存而只写入内存，
// 未真正落到磁盘。Sync 确保数据持久化，在程序紧接着崩溃时
// 不丢失数据。
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return dstFile.Sync()
}

// CopyDir 递归复制整个目录。
//
// 使用 filepath.Walk 遍历源目录的所有文件和子目录。
// 在目标路径下重建完整的目录结构，然后逐个复制文件。
//
// 参数：
//   - src: 源目录路径，必须存在。
//   - dst: 目标目录路径，不需要预先创建。
//
// 注意：软链接（symbolic link）不会被复制，而是复制其目标内容。
func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return CopyFile(path, dstPath)
	})
}

// FileLock 封装了跨平台文件锁。
//
// 底层使用 github.com/gofrs/flock，它在 Linux/macOS 上使用
// flock 系统调用，在 Windows 上使用 LockFileEx API。
//
// 文件锁的用途：
//   - 防止多进程同时写入同一个文件。
//   - 实现进程间互斥（例如只允许一个实例处理某个任务）。
//
// 注意：
//   - 文件锁在进程意外退出时会自动释放（操作系统负责清理）。
//   - 锁文件本身需要被创建（NewFileLock 不会创建文件，
//     锁定操作会隐式创建）。
//   - 文件锁是建议锁，不阻止其他进程直接读写文件（不强制）。
type FileLock struct {
	path  string       // 锁文件的路径
	flock *flock.Flock // 底层 flock 实例
}

// NewFileLock 创建指定路径的文件锁。
//
// 锁文件通常放在临时目录（如 /tmp/mylock.lock）。
// 不同业务使用不同的锁文件名。
//
// 使用示例：
//
//	lock := fileutil.NewFileLock("/tmp/myapp.lock")
//	lock.Lock()
//	defer lock.Unlock()
//	// ... 临界区代码 ...
func NewFileLock(path string) *FileLock {
	return &FileLock{
		path:  path,
		flock: flock.New(path),
	}
}

// Lock 以阻塞方式获取排它锁。
//
// 如果锁被其他进程持有，当前 goroutine 会一直阻塞等待直到获取到锁。
// 如果进程被信号中断，Lock 返回错误。
func (l *FileLock) Lock() error {
	return l.flock.Lock()
}

// Unlock 释放文件锁。
//
// 通常在 defer 中调用。如果进程在 Lock 后意外退出，
// 操作系统会自动释放锁，无需手动 Unlock。
func (l *FileLock) Unlock() error {
	return l.flock.Unlock()
}

// TryLock 尝试获取排它锁，不阻塞。
//
// 返回值：
//   - bool: true 表示成功获取锁，false 表示锁已被其他进程持有。
//   - error: 发生 I/O 错误时返回。
//
// 与 Lock 的区别：Lock 会一直等待直到成功，TryLock 立即返回。
func (l *FileLock) TryLock() (bool, error) {
	return l.flock.TryLock()
}
