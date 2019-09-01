package shutil

import "fmt"

type SameFileError struct {
	Src string
	Dst string
}

func (e SameFileError) Error() string {
	return fmt.Sprintf("%s and %s are the same file", e.Src, e.Dst)
}

type SpecialFileError struct {
	File string
}

func (e SpecialFileError) Error() string {
	return fmt.Sprintf("`%s` is a named pipe", e.File)
}

type CopyNotCompleteError struct {
	Src string
	Dst string
}

func (e CopyNotCompleteError) Error() string {
	return fmt.Sprintf("copy %s to %s not complete", e.Src, e.Dst)
}
