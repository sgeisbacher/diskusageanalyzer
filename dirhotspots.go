package main

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
)

type Dir struct {
	Name      string
	TotalSize int64
	Size      int64
	Children  []string
}

type DirsSizeDescSorter []*Dir
type DirsTotalSizeDescSorter []*Dir

func (dir DirsSizeDescSorter) Len() int           { return len(dir) }
func (dir DirsSizeDescSorter) Swap(i, j int)      { dir[i], dir[j] = dir[j], dir[i] }
func (dir DirsSizeDescSorter) Less(i, j int) bool { return dir[i].Size > dir[j].Size }

func (dir DirsTotalSizeDescSorter) Len() int           { return len(dir) }
func (dir DirsTotalSizeDescSorter) Swap(i, j int)      { dir[i], dir[j] = dir[j], dir[i] }
func (dir DirsTotalSizeDescSorter) Less(i, j int) bool { return dir[i].TotalSize > dir[j].TotalSize }

func (ctx *AnalyzerContext) AddFile(fileInfo FileInfo) {
	fileParent := filepath.Dir(fileInfo.Name)
	if fileParent == "." {
		return
	}
	dir, found := ctx.dirIdx[fileParent]
	if !found {
		fmt.Println("WARN: dir", fileParent, "NOT FOUND")
		return
	}
	dir.Size += fileInfo.Size
}

func (ctx *AnalyzerContext) AddDir(dir *Dir) {
	ctx.dirs = append(ctx.dirs, dir)
	ctx.dirIdx[dir.Name] = dir
	if dir.Name == ctx.root {
		return
	}
	parent, found := ctx.dirIdx[filepath.Dir(dir.Name)]
	if !found {
		fmt.Println("WARN: PARENT NOT FOUND:", filepath.Dir(dir.Name))
		return
	}
	parent.Children = append(parent.Children, dir.Name)
}

func (ctx *AnalyzerContext) getOrCreateDir(path string) *Dir {
	dir := ctx.dirIdx[path]
	if dir == nil {
		dir = &Dir{Name: path}
		ctx.dirIdx[path] = dir
		ctx.dirs = append(ctx.dirs, dir)
		parent, found := ctx.dirIdx[filepath.Dir(path)]
		if !found {
			fmt.Println("WARN: parent", filepath.Dir(path), "NOT FOUND")
			return dir
		}
		if parent.Children == nil {
			parent.Children = []string{}
		}
		parent.Children = append(parent.Children, path)
	}
	return dir
}

func (ctx *AnalyzerContext) GetDirHotspots(top int) Dirs {
	sort.Sort(DirsSizeDescSorter(ctx.dirs))
	limit := getLimit(len(ctx.dirs), top)
	return ctx.dirs[:limit]
}

func (ctx *AnalyzerContext) GetTreeHotspots(top int) Dirs {
	ctx.CalcTotalSizes()
	hotspots := ctx.dirs.Filter(isPotentialTreeHotspot(ctx, 0.8))

	sort.Sort(DirsTotalSizeDescSorter(hotspots))
	limit := getLimit(len(hotspots), top)
	return hotspots[:limit]
}

func isPotentialTreeHotspot(ctx *AnalyzerContext, threshold float64) DirFilter {
	return func(dir *Dir) bool {
		maxRelDiff := float64(0)
		if len(dir.Children) == 0 {
			return false
		}
		for _, childName := range dir.Children {
			child, found := ctx.dirIdx[childName]
			if !found {
				fmt.Printf("warn: child '%v' not found in index!!!!\n", childName)
				continue
			}
			relDiff := float64(child.TotalSize) / float64(dir.TotalSize)
			maxRelDiff = math.Max(maxRelDiff, relDiff)
			if maxRelDiff > threshold {
				return false
			}
		}
		return true
	}
}

type DirFilter func(dir *Dir) bool

func (vs Dirs) Filter(f DirFilter) Dirs {
	vsf := make(Dirs, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func getLimit(size int, top int) int {
	switch {
	case top <= 0:
		return size
	case top > 0 && top <= size:
		return top
	case top > size:
		return size
	}
	return 3
}
