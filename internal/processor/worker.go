package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorinha/internal/models"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var CACHE_PROCESSOR, CACHE_URL string
var CACHE_TIMESTAMP time.Time
var CACHE_TTL = 4.8

func getProcessor(ctx context.Context, client *redis.Client) (processor string, url string) {
	if CACHE_PROCESSOR != "" &&
		time.Since(CACHE_TIMESTAMP) < time.Duration(CACHE_TTL)*time.Second {
		fmt.Printf("[LOG] [getProcessor] Usando cache: %s -> %s\n", CACHE_PROCESSOR, CACHE_URL)
		return CACHE_PROCESSOR, CACHE_URL
	}

	data, err := client.Get(ctx, "processor").Result()
	if err != nil {
		fmt.Printf("[LOG] [getProcessor] Erro ao buscar 'processor' no Redis: %v\n", err)
		return "default", "http://payment-processor-default:8080"
	}

	fmt.Printf("[LOG] [getProcessor] Valor encontrado no Redis: '%s'\n", data)
	
	dt := strings.Split(data, "##")
	if len(dt) == 2 {
		processor = dt[0]
		url = dt[1]
		CACHE_PROCESSOR = processor
		CACHE_URL = url
		CACHE_TIMESTAMP = time.Now()
		fmt.Printf("[LOG] [getProcessor] Processador selecionado: %s -> %s\n", processor, url)
		return
	}

	fmt.Printf("[LOG] [getProcessor] Formato inesperado de valor no Redis: %v (len=%d). Usando default.\n", dt, len(dt))
	return "default", "http://payment-processor-default:8080"
}

func PostPayment(payment models.PaymentPost, url string) (requestedAt time.Time, err error) {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	requestedAt = time.Now().UTC()

	data := map[string]any{
		"correlationId": payment.CorrelationId,
		"amount":        payment.Amount,
		"requestedAt":   requestedAt,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("[LOG] [PostPayment] Erro ao serializar payload: %v\n", err)
		return
	}

	fmt.Printf("[LOG] [PostPayment] Enviando POST para %s com payload: %+v\n", url, data)

	res, err := httpClient.Post(
		fmt.Sprintf("%s/payments", url),
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		fmt.Printf("[LOG] [PostPayment] Erro na requisição: %v\n", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		fmt.Printf("[LOG] [PostPayment] Status inesperado: %d\n", res.StatusCode)
		return requestedAt, errors.New("erro na requisição para o processador")
	}

	fmt.Println("[LOG] [PostPayment] Requisição enviada com sucesso")
	return
}

func WorkerPayments(client *redis.Client) {
	ctx := context.Background()

	fmt.Println("[LOG] [WorkerPayments] Worker iniciado")

	for {
		res, err := client.BRPop(ctx, 0, "payment_queue").Result()
		if err != nil {
			fmt.Printf("[LOG] [WorkerPayments] Erro ao consumir fila: %v\n", err)
			continue
		}

		if len(res) < 2 {
			fmt.Println("[LOG] [WorkerPayments] Resultado inesperado da fila, pulando")
			continue
		}

		var p models.PaymentPost
		if err := json.Unmarshal([]byte(res[1]), &p); err != nil {
			fmt.Printf("[LOG] [WorkerPayments] Erro ao deserializar payload: %v - dado: %s\n", err, res[1])
			continue
		}

		fmt.Printf("[LOG] [WorkerPayments] Processando pagamento: %+v\n", p)

		processor, url := getProcessor(ctx, client)
		
		// Garantir que temos um processor válido
		if processor == "" {
			processor = "default"
			fmt.Printf("[LOG] [WorkerPayments] Processor vazio, usando default\n")
		}
		
		requestedAt, err := PostPayment(p, url)
		if err == nil {
			// Converter float32 para float64 para garantir consistência
			amount := float64(p.Amount)
			member := fmt.Sprintf(`%s#%.2f#%s`, p.CorrelationId, amount, processor)
			score := float64(requestedAt.Unix())

			fmt.Printf("[LOG] [WorkerPayments] Adicionando ao ZSET: member=%s, score=%f, processor=%s\n", member, score, processor)
			
			_, zErr := client.ZAdd(ctx, "payments", redis.Z{Member: member, Score: score}).Result()
			if zErr != nil {
				fmt.Printf("[LOG] [WorkerPayments] Erro ao adicionar no ZSET: %v\n", zErr)
			} else {
				fmt.Printf("[LOG] [WorkerPayments] Pagamento registrado com sucesso: %s (score=%f)\n", member, score)
			}
		} else {
			fmt.Printf("[LOG] [WorkerPayments] Erro ao processar pagamento: %v\n", err)
			time.Sleep(2 * time.Second)
		}
	}
}
