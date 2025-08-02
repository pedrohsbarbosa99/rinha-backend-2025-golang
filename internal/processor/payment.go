package processor

import (
	"context"
	"fmt"
	"gorinha/external/getway"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

func processPayment(ctx context.Context, c *redis.Client, payment models.PaymentPost) (requestedAt time.Time, processor string, err error) {
	fmt.Println("Processando pagamento:", payment.CorrelationId)
	requestedAt, err = getway.PostPayment(
		payment.Amount,
		payment.CorrelationId,
		env.PROCESSOR_DEFAULT_URL,
	)
	if err != nil {
		requestedAt, err = getway.PostPayment(
			payment.Amount,
			payment.CorrelationId,
			env.PROCESSOR_FALLBACK_URL,
		)
		if err == nil {
			database.AddPayment(
				ctx,
				c,
				payment.CorrelationId,
				"fallback",
				payment.Amount,
				requestedAt,
			)
		}
	} else {
		database.AddPayment(
			ctx,
			c,
			payment.CorrelationId,
			"default",
			payment.Amount,
			requestedAt,
		)
	}
	return
}

func processPayments(
	ctx context.Context,
	c *redis.Client,
	payments []models.PaymentPost,
	wg *sync.WaitGroup,
) {
	for _, p := range payments {
		payment := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := processPayment(ctx, c, payment)
			if err != nil {
				fmt.Println("Erro ao processar pagamento:", err)
			}
		}()
	}
}
