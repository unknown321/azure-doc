package main

import (
	"os"
	"reflect"
	"testing"
)

func TestMDFromYML(t *testing.T) {
	type args struct {
		ymlPath string
	}
	tests := []struct {
		name            string
		args            args
		wantMd          []byte
		wantContentPath string
		wantErr         bool
	}{
		{
			name:            "ok",
			args:            args{ymlPath: "tests/MDFromYML.yml"},
			wantMd:          []byte("hello"),
			wantContentPath: "tests/MDFromYML.md",
			wantErr:         false,
		},
		{
			name:            "ok in dir",
			args:            args{ymlPath: "tests/MDFromYML_inDir.yml"},
			wantMd:          []byte("qwe"),
			wantContentPath: "tests/dir1/MDFromYML_inDir.md",
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMd, gotContentPath, err := MDFromYML(tt.args.ymlPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("MDFromYML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotMd, tt.wantMd) {
				t.Errorf("MDFromYML() gotMd = %v, want %v", gotMd, tt.wantMd)
			}
			if gotContentPath != tt.wantContentPath {
				t.Errorf("MDFromYML() gotContentPath = %v, want %v", gotContentPath, tt.wantContentPath)
			}
		})
	}
}

func TestImageFromTripleColon(t *testing.T) {
	type args struct {
		f []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "one image",
			args: args{f: []byte(`:::image type="complex" source="./images/private-link-hub-spoke-network-private-link.png" alt-text="alt." border="false":::`)},
			want: []byte(`![alt.](./images/private-link-hub-spoke-network-private-link.png)`),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImageFromTripleColon(tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImageFromTripleColon() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFixMDImages(t *testing.T) {
	type args struct {
		f        string
		fullPath string
		basePath string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "one image, ok",
			args: args{
				f:        "tests/imgFix/imgFix1.md",
				fullPath: "fp",
				basePath: "bp",
			},
			want: []byte("(../images/qwe_1_2_3.png)\n" +
				"![1 (sharding) key](../images/datapartitioning01.png)\n" +
				"[title]: ../images/_images_adas.jpg\n" +
				"();\n}\n```\n\n" +
				"![title](../images/file.png)\n" +
				"![title](../images/example.png \"subtitle\")\n" +
				"(../images/test.png)\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data []byte
			var err error
			if data, err = os.ReadFile(tt.args.f); err != nil {
				t.Fatalf("cannot read test data: %s", err.Error())
			}
			if got := FixMDImages(data, tt.args.fullPath, tt.args.basePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FixMDImages() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFixHTMLImages(t *testing.T) {
	type args struct {
		f        string
		fullPath string
		basePath string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "one",
			args: args{
				f:        "tests/imgFix/imgFixHTML.html",
				fullPath: "fp",
				basePath: "bp",
			},
			want: []byte("<img role=\"p\" alt=\"alt\" src=\"../images/reference-architectures_hybrid-networking_images_shared-services.png\">\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data []byte
			var err error
			if data, err = os.ReadFile(tt.args.f); err != nil {
				t.Fatalf("cannot read test data: %s", err.Error())
			}
			if got := FixHTMLImages(data, tt.args.fullPath, tt.args.basePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FixHTMLImages() =\n%s\n======\nwant\n%s\n", got, tt.want)
			}
		})
	}
}
