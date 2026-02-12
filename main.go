package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Конфигурация теста
const (
	TotalRequests      = 2000 // Общее количество запросов
	ConcurrentRequests = 1000 // Количество одновременных запросов
	RequestTimeout     = 30   // Таймаут запроса в секундах
	BaseURL            = "http://localhost:3000"
	HealthEndpoint     = "/health"
	TaskEndpoint       = "/v1/users/ecd3ba23-d60d-46f1-97e6-34b4693b3363/tasks"
)

// JWT токен (в реальном приложении должен быть в переменных окружения)
const AuthToken = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzA4NDA2NDcsImlhdCI6MTc3MDgzOTc0NywianRpIjoiMWFkMDY2OWMtNDcyMi00YjM5LWEzZDMtYjRhZDZhOWJiMDg4Iiwicm9sZXMiOiJ1c2VyIiwic2Vzc2lvbiI6ImVjZDNiYTIzLWQ2MGQtNDZmMS05N2U2LTM0YjQ2OTNiMzM2MzoxMjMiLCJzdWIiOiJlY2QzYmEyMy1kNjBkLTQ2ZjEtOTdlNi0zNGI0NjkzYjMzNjMiLCJ2ZXIiOjF9.xX-yrbI0XT6ZprR1YxL44_SRIngNW9hcZwT2qkwY1oI"

// Статистика
type Statistics struct {
	TotalRequests      int64         // Общее количество запросов
	SuccessfulRequests int64         // Успешные запросы
	FailedRequests     int64         // Неудачные запросы
	TotalResponseTime  int64         // Общее время ответов в наносекундах
	MinResponseTime    int64         // Минимальное время ответа
	MaxResponseTime    int64         // Максимальное время ответа
	StatusCodes        map[int]int64 // Распределение статус-кодов
	StartTime          time.Time
	EndTime            time.Time
}

// Глобальные переменные
var (
	stats       Statistics
	statsMutex  sync.RWMutex
	client      *http.Client
	wg          sync.WaitGroup
	sem         chan struct{} // Семафор для ограничения конкурентности
	taskCounter int32         // Счетчик для уникальных ID задач
)

func init() {
	// Инициализация статистики
	stats.StatusCodes = make(map[int]int64)
	stats.MinResponseTime = 1<<63 - 1 // Максимальное значение int64

	// Инициализация HTTP клиента
	client = &http.Client{
		Timeout: RequestTimeout * time.Second,
		Transport: &http.Transport{
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          ConcurrentRequests * 2,
			MaxIdleConnsPerHost:   ConcurrentRequests,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		},
	}

	// Инициализация семафора для контроля конкурентности
	sem = make(chan struct{}, ConcurrentRequests)
}

func main() {
	log.Println("🚀 НАГРУЗОЧНЫЙ ТЕСТ ЗАДАЧ")
	log.Println("═══════════════════════════════════════════")
	log.Printf("Общее количество запросов: %d", TotalRequests)
	log.Printf("Конкурентных запросов: %d", ConcurrentRequests)
	log.Printf("URL: %s%s", BaseURL, TaskEndpoint)
	log.Println("═══════════════════════════════════════════")

	// Проверяем доступность сервиса
	if !checkHealth() {
		log.Fatal("❌ Сервис недоступен, тест прерван")
	}

	// Запускаем нагрзочный тест
	stats.StartTime = time.Now()
	runLoadTest()
	stats.EndTime = time.Now()

	// Выводим статистику
	printStatistics()

	// Тест завершен
	log.Println("✅ Тест завершен")
}

func checkHealth() bool {
	log.Println("🔍 Проверка доступности сервиса...")

	resp, err := client.Get(BaseURL + HealthEndpoint)
	if err != nil {
		log.Printf("❌ Сервис недоступен: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Health check вернул статус: %d", resp.StatusCode)
		return false
	}

	log.Println("✅ Сервис доступен")
	return true
}

func runLoadTest() {
	log.Println("▶️ Запуск нагрузочного теста...")

	// Создаем канал для отправки номеров запросов в worker'ы
	requests := make(chan int, TotalRequests)

	// Заполняем канал номерами запросов
	go func() {
		for i := 1; i <= TotalRequests; i++ {
			requests <- i
		}
		close(requests)
	}()

	// Запускаем worker'ы
	for i := 0; i < ConcurrentRequests; i++ {
		wg.Add(1)
		go worker(i+1, requests)
	}

	// Ждем завершения всех worker'ов
	wg.Wait()
	log.Println("✅ Все запросы обработаны")
}

func worker(id int, requests <-chan int) {
	defer wg.Done()

	for reqNum := range requests {
		// Ограничиваем конкурентность с помощью семафора
		sem <- struct{}{}

		// Выполняем запрос
		start := time.Now()
		statusCode, err := sendTaskRequest(reqNum)
		duration := time.Since(start)

		// Освобождаем семафор
		<-sem

		// Обновляем статистику
		updateStats(statusCode, err, duration)

		// Выводим прогресс каждые 10 запросов
		if reqNum%10 == 0 {
			log.Printf("👷 Worker %d: обработал запрос %d/%d (статус: %d, время: %v)",
				id, reqNum, TotalRequests, statusCode, duration)
		}
	}
}

func sendTaskRequest(requestNum int) (int, error) {
	// Генерируем уникальный ID для задачи
	taskID := atomic.AddInt32(&taskCounter, 1)
	due := time.Now().Add(48 * time.Hour).Format("2006-01-02 15:04") // дата через 2 дня

	// Создаем тело запроса с уникальными данными
	body := fmt.Sprintf(`{
	"title": "Тестовая задача #%d (запрос %d)",
	"description": "Автоматически созданная задача из нагрузочного теста",
	"status": "1",
	"priory": "blue",
	"due_date": "%s"
}`, taskID, requestNum, due)

	req, err := http.NewRequest("POST", BaseURL+TaskEndpoint, bytes.NewBufferString(body))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", AuthToken)
	req.Header.Set("User-Agent", "LoadTest/1.0")
	req.Header.Set("X-Request-ID", fmt.Sprintf("load-test-%d-%d", time.Now().Unix(), requestNum))

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func updateStats(statusCode int, err error, duration time.Duration) {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	atomic.AddInt64(&stats.TotalRequests, 1)

	durationNs := duration.Nanoseconds()
	stats.TotalResponseTime += durationNs

	// Обновляем минимальное время
	if durationNs < stats.MinResponseTime {
		stats.MinResponseTime = durationNs
	}

	// Обновляем максимальное время
	if durationNs > stats.MaxResponseTime {
		stats.MaxResponseTime = durationNs
	}

	if err != nil || statusCode >= 400 {
		atomic.AddInt64(&stats.FailedRequests, 1)
	} else {
		atomic.AddInt64(&stats.SuccessfulRequests, 1)
	}

	// Учитываем статус код
	stats.StatusCodes[statusCode]++
}

func printStatistics() {
	statsMutex.RLock()
	defer statsMutex.RUnlock()

	log.Println("\n═══════════════════════════════════════════")
	log.Println("📊 СТАТИСТИКА ВЫПОЛНЕНИЯ ТЕСТА")
	log.Println("═══════════════════════════════════════════")

	totalTime := stats.EndTime.Sub(stats.StartTime).Seconds()
	rps := float64(stats.TotalRequests) / totalTime

	fmt.Printf("Общее время теста: %.2f секунд\n", totalTime)
	fmt.Printf("Всего запросов: %d\n", stats.TotalRequests)
	fmt.Printf("Успешных запросов: %d (%.1f%%)\n",
		stats.SuccessfulRequests,
		float64(stats.SuccessfulRequests)/float64(stats.TotalRequests)*100)
	fmt.Printf("Неудачных запросов: %d (%.1f%%)\n",
		stats.FailedRequests,
		float64(stats.FailedRequests)/float64(stats.TotalRequests)*100)
	fmt.Printf("RPS (запросов в секунду): %.2f\n", rps)

	if stats.TotalRequests > 0 {
		avgResponseTime := time.Duration(stats.TotalResponseTime / stats.TotalRequests)
		minResponseTime := time.Duration(stats.MinResponseTime)
		maxResponseTime := time.Duration(stats.MaxResponseTime)

		fmt.Printf("Среднее время ответа: %v\n", avgResponseTime)
		fmt.Printf("Минимальное время ответа: %v\n", minResponseTime)
		fmt.Printf("Максимальное время ответа: %v\n", maxResponseTime)
	}

	fmt.Println("\n📈 Распределение статус-кодов:")
	for code, count := range stats.StatusCodes {
		percentage := float64(count) / float64(stats.TotalRequests) * 100
		fmt.Printf("  %d: %d запросов (%.1f%%)\n", code, count, percentage)
	}

	// Вычисляем перцентили
	printPercentiles()

	fmt.Println("\n⚙️ Конфигурация теста:")
	fmt.Printf("  Общее количество запросов: %d\n", TotalRequests)
	fmt.Printf("  Конкурентных запросов: %d\n", ConcurrentRequests)
	fmt.Printf("  Таймаут запроса: %d секунд\n", RequestTimeout)
	fmt.Printf("  URL: %s%s\n", BaseURL, TaskEndpoint)

	log.Println("═══════════════════════════════════════════")
}

func printPercentiles() {
	// В реальном приложении здесь нужно хранить все времена ответов
	// Для упрощения выводим только расчетные значения
	if stats.TotalRequests > 0 {
		avg := time.Duration(stats.TotalResponseTime / stats.TotalRequests)
		min := time.Duration(stats.MinResponseTime)
		max := time.Duration(stats.MaxResponseTime)

		// Оценочные перцентили (в реальном приложении нужно считать точно)
		fmt.Println("\n📊 Оценочные перцентили времени ответа:")
		fmt.Printf("  P50 (медиана): ~%v\n", avg)
		fmt.Printf("  P90: ~%v\n", time.Duration(float64(avg.Nanoseconds())*1.3))
		fmt.Printf("  P95: ~%v\n", time.Duration(float64(avg.Nanoseconds())*1.5))
		fmt.Printf("  P99: ~%v\n", time.Duration(float64(avg.Nanoseconds())*1.8))
		fmt.Printf("  Min: %v\n", min)
		fmt.Printf("  Max: %v\n", max)
	}
}

// Для более точной статистики можно использовать эту структуру:
type RequestResult struct {
	Duration   time.Duration
	StatusCode int
	Error      error
}

// И хранить все результаты в слайсе для расчета перцентилей
