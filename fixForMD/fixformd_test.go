package fixForMD

import "testing"

func TestRewritePath(t *testing.T) {
	type args struct {
		imagePath   string
		contentPath string
		basePath    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1",
			args: args{imagePath: "./image.png", contentPath: "./content/data.md", basePath: ""},
			want: "content/image.png",
		},
		{
			name: "",
			args: args{
				imagePath:   "../../image.png",
				contentPath: "./test/1/2/3/data.md",
				basePath:    "",
			},
			want: "test/1/image.png",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RewritePath(tt.args.imagePath, tt.args.contentPath, tt.args.basePath); got != tt.want {
				t.Errorf("RewritePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
