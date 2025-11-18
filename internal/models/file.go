package models

type FileStatus string

const (
	StatusModified  FileStatus = "M"  // Modified
	StatusAdded     FileStatus = "A"  // Added (staged new file)
	StatusDeleted   FileStatus = "D"  // Deleted
	StatusRenamed   FileStatus = "R"  // Renamed
	StatusCopied    FileStatus = "C"  // Copied
	StatusUntracked FileStatus = "??" // Untracked
	StatusUpdated   FileStatus = "U"  // Updated but unmerged
)

type FileChange struct {
	Path          string
	Status        FileStatus     // Working tree status
	StagedStatus  FileStatus     // Staging area status
	IsStaged      bool
	IsUntracked   bool
}

func (f *FileChange) DisplayStatus() string {
	if f.IsUntracked {
		return "??"
	}

	staged := " "
	working := " "

	if f.StagedStatus != "" {
		staged = string(f.StagedStatus)
	}
	if f.Status != "" {
		working = string(f.Status)
	}

	return staged + working
}
