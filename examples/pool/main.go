package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	email "github.com/KARTIKrocks/goemail"
)

func main() {
	// Configure SMTP with connection pooling
	config := email.SMTPConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     587,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
		UseTLS:   true,

		// Pool configuration
		PoolSize:        5,                // Up to 5 concurrent SMTP connections
		MaxIdleConns:    2,                // Keep 2 idle connections warm
		PoolMaxLifetime: 30 * time.Minute, // Rotate connections after 30 minutes
		PoolMaxIdleTime: 5 * time.Minute,  // Evict idle connections after 5 minutes
		MaxMessages:     100,              // Rotate after 100 messages per connection
		PoolWaitTimeout: 10 * time.Second, // Wait up to 10s for a connection
	}

	sender, err := email.NewSMTPSender(config)
	if err != nil {
		log.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close() // Important: closes all pooled connections

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Send 50 emails concurrently using the pool
	recipients := make([]string, 50)
	for i := range recipients {
		recipients[i] = fmt.Sprintf("user%d@example.com", i+1)
	}

	log.Printf("Sending %d emails with pool size %d...", len(recipients), config.PoolSize)
	start := time.Now()

	var wg sync.WaitGroup
	var errCount int
	var mu sync.Mutex

	for i, recipient := range recipients {
		wg.Add(1)
		go func() {
			defer wg.Done()

			e := email.NewEmail().
				SetFrom(config.From).
				AddTo(recipient).
				SetSubject(fmt.Sprintf("Pooled Email #%d", i+1)).
				SetBody(fmt.Sprintf("This is email %d sent via connection pool.", i+1))

			if sendErr := sender.Send(ctx, e); sendErr != nil {
				mu.Lock()
				errCount++
				mu.Unlock()
				log.Printf("Failed to send to %s: %v", recipient, sendErr)
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	sent := len(recipients) - errCount
	log.Printf("Sent %d/%d emails in %s (%.1f emails/sec)",
		sent, len(recipients), duration, float64(sent)/duration.Seconds())
}
