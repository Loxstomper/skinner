package git

import (
	"testing"
	"time"
)

func TestParseLogOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Commit
		wantErr bool
	}{
		{
			name: "single commit with stats",
			input: "---COMMIT_SEP---\n" +
				"a3f2c1b\n" +
				"Fix parser edge case\n" +
				"2024-01-15T10:30:00-05:00\n" +
				"\n" +
				"12\t3\tmain.go\n" +
				"8\t4\tparser.go\n",
			want: []Commit{
				{
					Hash:       "a3f2c1b",
					Subject:    "Fix parser edge case",
					AuthorDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.FixedZone("", -5*3600)),
					Additions:  20,
					Deletions:  7,
				},
			},
		},
		{
			name: "multiple commits",
			input: "---COMMIT_SEP---\n" +
				"a3f2c1b\n" +
				"Fix parser\n" +
				"2024-01-15T10:30:00-05:00\n" +
				"\n" +
				"12\t3\tmain.go\n" +
				"\n" +
				"---COMMIT_SEP---\n" +
				"b1c4d5e\n" +
				"Add feature\n" +
				"2024-01-14T09:00:00-05:00\n" +
				"\n" +
				"5\t2\tfeature.go\n" +
				"3\t0\ttest.go\n",
			want: []Commit{
				{
					Hash:       "a3f2c1b",
					Subject:    "Fix parser",
					AuthorDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.FixedZone("", -5*3600)),
					Additions:  12,
					Deletions:  3,
				},
				{
					Hash:       "b1c4d5e",
					Subject:    "Add feature",
					AuthorDate: time.Date(2024, 1, 14, 9, 0, 0, 0, time.FixedZone("", -5*3600)),
					Additions:  8,
					Deletions:  2,
				},
			},
		},
		{
			name: "commit with no file changes",
			input: "---COMMIT_SEP---\n" +
				"c1d2e3f\n" +
				"Merge branch\n" +
				"2024-01-13T08:00:00Z\n",
			want: []Commit{
				{
					Hash:       "c1d2e3f",
					Subject:    "Merge branch",
					AuthorDate: time.Date(2024, 1, 13, 8, 0, 0, 0, time.UTC),
					Additions:  0,
					Deletions:  0,
				},
			},
		},
		{
			name:  "empty output",
			input: "",
			want:  nil,
		},
		{
			name: "binary files in numstat are skipped",
			input: "---COMMIT_SEP---\n" +
				"d4e5f6a\n" +
				"Add image\n" +
				"2024-01-12T07:00:00Z\n" +
				"\n" +
				"-\t-\timage.png\n" +
				"5\t2\tREADME.md\n",
			want: []Commit{
				{
					Hash:       "d4e5f6a",
					Subject:    "Add image",
					AuthorDate: time.Date(2024, 1, 12, 7, 0, 0, 0, time.UTC),
					Additions:  5,
					Deletions:  2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLogOutput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseLogOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseLogOutput() got %d commits, want %d", len(got), len(tt.want))
			}
			for i, want := range tt.want {
				if got[i].Hash != want.Hash {
					t.Errorf("commit[%d].Hash = %q, want %q", i, got[i].Hash, want.Hash)
				}
				if got[i].Subject != want.Subject {
					t.Errorf("commit[%d].Subject = %q, want %q", i, got[i].Subject, want.Subject)
				}
				if !got[i].AuthorDate.Equal(want.AuthorDate) {
					t.Errorf("commit[%d].AuthorDate = %v, want %v", i, got[i].AuthorDate, want.AuthorDate)
				}
				if got[i].Additions != want.Additions {
					t.Errorf("commit[%d].Additions = %d, want %d", i, got[i].Additions, want.Additions)
				}
				if got[i].Deletions != want.Deletions {
					t.Errorf("commit[%d].Deletions = %d, want %d", i, got[i].Deletions, want.Deletions)
				}
			}
		})
	}
}

func TestParseDiffTreeOutput(t *testing.T) {
	tests := []struct {
		name       string
		numstat    string
		nameStatus string
		want       []FileChange
	}{
		{
			name:       "modified and added files",
			numstat:    "12\t3\tmain.go\n34\t0\tnewfile.go\n0\t28\told.go\n",
			nameStatus: "M\tmain.go\nA\tnewfile.go\nD\told.go\n",
			want: []FileChange{
				{Status: "M", Path: "main.go", Additions: 12, Deletions: 3},
				{Status: "A", Path: "newfile.go", Additions: 34, Deletions: 0},
				{Status: "D", Path: "old.go", Additions: 0, Deletions: 28},
			},
		},
		{
			name:       "rename with score",
			numstat:    "5\t3\tnewname.go\n",
			nameStatus: "R100\toldname.go\tnewname.go\n",
			want: []FileChange{
				{Status: "R", Path: "newname.go", Additions: 5, Deletions: 3},
			},
		},
		{
			name:       "binary file",
			numstat:    "-\t-\timage.png\n",
			nameStatus: "A\timage.png\n",
			want: []FileChange{
				{Status: "A", Path: "image.png", Additions: 0, Deletions: 0},
			},
		},
		{
			name:       "empty output",
			numstat:    "",
			nameStatus: "",
			want:       nil,
		},
		{
			name:       "single file modification",
			numstat:    "1\t1\tREADME.md\n",
			nameStatus: "M\tREADME.md\n",
			want: []FileChange{
				{Status: "M", Path: "README.md", Additions: 1, Deletions: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDiffTreeOutput(tt.numstat, tt.nameStatus)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseDiffTreeOutput() got %d changes, want %d", len(got), len(tt.want))
			}
			for i, want := range tt.want {
				if got[i].Status != want.Status {
					t.Errorf("change[%d].Status = %q, want %q", i, got[i].Status, want.Status)
				}
				if got[i].Path != want.Path {
					t.Errorf("change[%d].Path = %q, want %q", i, got[i].Path, want.Path)
				}
				if got[i].Additions != want.Additions {
					t.Errorf("change[%d].Additions = %d, want %d", i, got[i].Additions, want.Additions)
				}
				if got[i].Deletions != want.Deletions {
					t.Errorf("change[%d].Deletions = %d, want %d", i, got[i].Deletions, want.Deletions)
				}
			}
		})
	}
}

func TestSumNumstat(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantAdditions int
		wantDeletions int
	}{
		{
			name:          "multiple files",
			input:         "12\t3\tmain.go\n8\t4\tparser.go\n",
			wantAdditions: 20,
			wantDeletions: 7,
		},
		{
			name:          "binary files skipped",
			input:         "-\t-\timage.png\n5\t2\tREADME.md\n",
			wantAdditions: 5,
			wantDeletions: 2,
		},
		{
			name:          "empty input",
			input:         "",
			wantAdditions: 0,
			wantDeletions: 0,
		},
		{
			name:          "blank lines ignored",
			input:         "\n\n3\t1\tfile.go\n\n",
			wantAdditions: 3,
			wantDeletions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA, gotD := sumNumstat(tt.input)
			if gotA != tt.wantAdditions {
				t.Errorf("sumNumstat() additions = %d, want %d", gotA, tt.wantAdditions)
			}
			if gotD != tt.wantDeletions {
				t.Errorf("sumNumstat() deletions = %d, want %d", gotD, tt.wantDeletions)
			}
		})
	}
}

func TestNonEmptyLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"with blanks", "a\n\nb\n\nc\n", 3},
		{"only blanks", "\n\n\n", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nonEmptyLines(tt.input)
			if len(got) != tt.want {
				t.Errorf("nonEmptyLines() got %d lines, want %d", len(got), tt.want)
			}
		})
	}
}
