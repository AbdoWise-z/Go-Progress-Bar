package progress_bar

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Default values
const DefaultFormat = "%prog%% |%bar%| %curr%/%max% [%eta%, %spd%it/s]"
const DefaultBarChar = "■"
const DefaultBarSize = 80
const DefaultStep = 1

type ProgressBar struct {
	barFormat   string
	progress    int64
	maxProgress int64
	barSize     int
	barChar     string
	speed       float64
	startTime   time.Time
	gamma       float64
}

func NewProgressBar(format string, max int64, size int, char string) *ProgressBar {
	if max < 1 {
		max = 1
	}
	if format == "" {
		format = DefaultFormat
	}
	if char == "" {
		char = DefaultBarChar
	}
	return &ProgressBar{
		barFormat:   format,
		maxProgress: max,
		barSize:     size,
		barChar:     char,
		startTime:   time.Now(),
		gamma:       0.5,
		speed:       0,
		progress:    0,
	}
}

func customFormat(format string, replacements map[string]string) string {
	for key, value := range replacements {
		format = strings.ReplaceAll(format, "%"+key+"%", value)
	}
	return strings.ReplaceAll(format, "\\%", "%")
}

func (pb *ProgressBar) render() {
	timeRemaining := float64(pb.maxProgress-pb.progress) / max(pb.speed, 0.01)

	speedStr := fmt.Sprintf("%.2f", pb.speed)
	timeStr := fmt.Sprintf("%02d:%05.2f", int(timeRemaining/60), float64(int(timeRemaining)%60)+timeRemaining-float64(int(timeRemaining)))
	currStr := fmt.Sprintf("%d", pb.progress)
	maxStr := fmt.Sprintf("%d", pb.maxProgress)
	progressStr := fmt.Sprintf("%.2f", 100.0*float64(pb.progress)/float64(pb.maxProgress))

	replacements := map[string]string{
		"spd":  speedStr,
		"eta":  timeStr,
		"curr": currStr,
		"max":  maxStr,
		"prog": progressStr,
	}

	barStr := customFormat(pb.barFormat, replacements)

	if strings.Contains(barStr, "%bar%") {
		filled := int(float64(pb.barSize) * float64(pb.progress) / float64(pb.maxProgress))
		bar := strings.Repeat(pb.barChar, filled) + strings.Repeat(" ", pb.barSize-filled)
		barStr = strings.ReplaceAll(barStr, "%bar%", bar)
	}

	fmt.Printf("\r%s", barStr)
	os.Stdout.Sync()
}

func (pb *ProgressBar) Step() {
	pb.SetProgress(pb.progress + 1)
}

func (pb *ProgressBar) SetProgress(prog int64) {
	duration := time.Since(pb.startTime).Seconds()
	if duration <= 1e-6 {
		duration = 1e-6
	}

	pb.startTime = time.Now()

	if prog < 0 {
		prog = 0
	} else if prog > pb.maxProgress {
		prog = pb.maxProgress
	}
	delta := float64(prog - pb.progress)
	pb.speed = pb.speed + pb.gamma*(delta/duration-pb.speed)

	pb.progress = prog
	pb.render()
}

func (pb *ProgressBar) Reset() {
	pb.progress = 0
	pb.speed = 0
	pb.startTime = time.Now()
}

type ProgressIterator[T any] struct {
	iter     []T
	index    int
	progress *ProgressBar
}

func NewProgressIterator[T any](iter []T, progress *ProgressBar) *ProgressIterator[T] {
	return &ProgressIterator[T]{iter: iter, index: 0, progress: progress}
}

func (it *ProgressIterator[T]) Next() (T, bool) {
	if it.index >= len(it.iter) {
		var zero T
		return zero, false
	}
	value := it.iter[it.index]
	it.index++
	it.progress.SetProgress(int64(it.index))
	return value, true
}

func (it *ProgressIterator[T]) Reset() {
	it.index = 0
	it.progress.Reset()
}

type RangeIteratorImpl struct {
	start, end, step, current int
	progress                  *ProgressBar
}

func NewRangeIterator(start, end, step int, format string, size int, char string) *RangeIteratorImpl {
	if step == 0 {
		step = DefaultStep
	}
	pb := NewProgressBar(format, int64((end-start)/step), size, char)
	return &RangeIteratorImpl{start: start, end: end, step: step, current: start, progress: pb}
}

func RangeIterator(start, end int) *RangeIteratorImpl {
	return NewRangeIterator(start, end, DefaultStep, DefaultFormat, DefaultBarSize, DefaultBarChar)
}

func StepRangeIterator(start, end, step int) *RangeIteratorImpl {
	return NewRangeIterator(start, end, step, DefaultFormat, DefaultBarSize, DefaultBarChar)
}

func (ri *RangeIteratorImpl) Next() (int, bool) {
	if (ri.step > 0 && ri.current >= ri.end) || (ri.step < 0 && ri.current <= ri.end) {
		return 0, false
	}
	val := ri.current
	ri.current += ri.step
	ri.progress.Step()
	return val, true
}

func (it *ProgressIterator[T]) Iter() <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		for {
			value, ok := it.Next()
			if !ok {
				break
			}
			ch <- value
		}
	}()
	return ch
}

func (ri *RangeIteratorImpl) Iter() <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for {
			value, ok := ri.Next()
			if !ok {
				break
			}
			ch <- value
		}
	}()
	return ch
}
