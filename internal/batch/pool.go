package batch

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

// Job bir dönüşüm işini temsil eder
type Job struct {
	InputPath  string
	OutputPath string
	From       string
	To         string
	Options    converter.Options
	SkipReason string
}

// JobResult bir işin sonucunu tutar
type JobResult struct {
	Job        Job
	Success    bool
	Skipped    bool
	Attempts   int
	OutputSize int64
	SkipReason string
	Error      error
	Duration   time.Duration
}

// Pool worker pool'u yönetir
type Pool struct {
	Workers    int
	RetryMax   int
	RetryDelay time.Duration
	Results    []JobResult
	mu         sync.Mutex
	processed  atomic.Int64
	totalJobs  int
	OnProgress func(completed, total int) // İlerleme callback'i
}

// NewPool yeni bir worker pool oluşturur
func NewPool(workers int) *Pool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	// Çok fazla worker açmayı engelle
	maxWorkers := runtime.NumCPU() * 2
	if workers > maxWorkers {
		workers = maxWorkers
	}

	return &Pool{
		Workers:    workers,
		RetryDelay: 500 * time.Millisecond,
	}
}

// SetRetry retry davranışını ayarlar.
func (p *Pool) SetRetry(max int, delay time.Duration) {
	if max < 0 {
		max = 0
	}
	p.RetryMax = max

	if delay >= 0 {
		p.RetryDelay = delay
	}
}

// Execute verilen işleri paralel olarak çalıştırır
func (p *Pool) Execute(jobs []Job) []JobResult {
	p.totalJobs = len(jobs)
	p.Results = make([]JobResult, 0, len(jobs))
	p.processed.Store(0)

	if len(jobs) == 0 {
		return p.Results
	}

	// Worker sayısını iş sayısına göre ayarla
	workers := p.Workers
	if workers > len(jobs) {
		workers = len(jobs)
	}

	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan JobResult, len(jobs))

	// Worker'ları başlat
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobChan {
				result := p.processJob(job)
				resultChan <- result
			}
		}()
	}

	// İşleri gönder
	go func() {
		for _, job := range jobs {
			jobChan <- job
		}
		close(jobChan)
	}()

	// Sonuçları topla
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Sonuçları oku ve ilerleme bildir
	for result := range resultChan {
		p.mu.Lock()
		p.Results = append(p.Results, result)
		p.mu.Unlock()

		completed := int(p.processed.Add(1))
		if p.OnProgress != nil {
			p.OnProgress(completed, p.totalJobs)
		}
	}

	return p.Results
}

// processJob tek bir dönüşüm işini gerçekleştirir
func (p *Pool) processJob(job Job) JobResult {
	start := time.Now()

	if job.SkipReason != "" {
		return JobResult{
			Job:        job,
			Skipped:    true,
			Success:    false,
			SkipReason: job.SkipReason,
			Duration:   time.Since(start),
		}
	}

	// Converter bul
	conv, err := converter.FindConverter(job.From, job.To)
	if err != nil {
		return JobResult{
			Job:      job,
			Success:  false,
			Attempts: 1,
			Error:    err,
			Duration: time.Since(start),
		}
	}

	// Çıktı dizinini oluştur
	outputDir := filepath.Dir(job.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return JobResult{
			Job:      job,
			Success:  false,
			Attempts: 1,
			Error:    fmt.Errorf("çıktı dizini oluşturulamadı: %w", err),
			Duration: time.Since(start),
		}
	}

	var lastErr error
	attempts := p.RetryMax + 1
	if attempts <= 0 {
		attempts = 1
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		// Dönüşümü yap
		err = conv.Convert(job.InputPath, job.OutputPath, job.Options)
		if err == nil {
			size := int64(0)
			if info, statErr := os.Stat(job.OutputPath); statErr == nil {
				size = info.Size()
			}
			return JobResult{
				Job:        job,
				Success:    true,
				Attempts:   attempt,
				OutputSize: size,
				Duration:   time.Since(start),
			}
		}

		lastErr = err
		if attempt < attempts && p.RetryDelay > 0 {
			time.Sleep(p.RetryDelay)
		}
	}

	return JobResult{
		Job:      job,
		Success:  false,
		Attempts: attempts,
		Error:    lastErr,
		Duration: time.Since(start),
	}
}

// Summary toplu iş sonuçlarını özetler
type Summary struct {
	Total     int
	Succeeded int
	Skipped   int
	Failed    int
	Duration  time.Duration
	Errors    []JobError
}

// JobError başarısız olan bir işin hata bilgisi
type JobError struct {
	InputFile string
	Error     string
	Attempts  int
}

// GetSummary iş sonuçlarından özet oluşturur
func GetSummary(results []JobResult, totalDuration time.Duration) Summary {
	s := Summary{
		Total:    len(results),
		Duration: totalDuration,
	}

	for _, r := range results {
		if r.Success {
			s.Succeeded++
		} else if r.Skipped {
			s.Skipped++
		} else {
			s.Failed++
			msg := "bilinmeyen hata"
			if r.Error != nil {
				msg = r.Error.Error()
			}
			s.Errors = append(s.Errors, JobError{
				InputFile: r.Job.InputPath,
				Error:     msg,
				Attempts:  r.Attempts,
			})
		}
	}

	return s
}

// CollectFiles dizindeki belirli uzantıya sahip dosyaları toplar
func CollectFiles(dir string, fromFormat string, recursive bool) ([]string, error) {
	var files []string

	walkFn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Erişilemeyen dosyaları atla
		}

		if d.IsDir() {
			// Recursive değilse alt dizinlere girme
			if !recursive && path != dir {
				return filepath.SkipDir
			}
			return nil
		}

		if converter.HasFormatExtension(path, fromFormat) {
			files = append(files, path)
		}

		return nil
	}

	if err := filepath.WalkDir(dir, walkFn); err != nil {
		return nil, fmt.Errorf("dizin taranamadı: %w", err)
	}

	return files, nil
}

// CollectFilesFromGlob glob pattern ile dosya toplar
func CollectFilesFromGlob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern hatası: %w", err)
	}

	var files []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			files = append(files, m)
		}
	}

	return files, nil
}
