package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/machbase/neo-client/v2"
	"github.com/machbase/neo-client/v2/api"
)

var (
	host        = "127.0.0.1"
	port        = 5656
	user        = "sys"
	password    = "manager"
	keyDir      = "/tmp/authaging_keys"
	keyName     = "authaging"
	concurrency = 1
	verbose     = false
)

func main() {
	flag.StringVar(&host, "h", host, "server host")
	flag.IntVar(&port, "p", port, "server port")
	flag.StringVar(&user, "u", user, "user")
	flag.StringVar(&password, "P", password, "password")
	flag.StringVar(&keyDir, "dir", keyDir, "directory to store generated auth key files")
	flag.StringVar(&keyName, "name", keyName, "auth key file name prefix")
	flag.IntVar(&concurrency, "c", concurrency, "number of concurrent loops")
	flag.BoolVar(&verbose, "v", verbose, "enable verbose logging")
	flag.Parse()
	if concurrency < 1 {
		concurrency = 1
	}

	ctx := context.Background()

	user = strings.ToUpper(user)

	db, err := sql.Open("machbase", fmt.Sprintf("host=%s;port=%d;user=%s;password=%s", host, port, user, password))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		panic(err)
	}

	privatePath, publicPEM, err := ensureAuthKeyPair(keyDir, keyName)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Using auth key files in %s\n", keyDir)
	fmt.Printf("Private key: %s\n", privatePath)

	adminConn, err := db.Conn(ctx)
	if err != nil {
		panic(err)
	}

	if err := registerAuthKey(ctx, adminConn, user, publicPEM); err != nil {
		adminConn.Close()
		panic(err)
	}
	adminConn.Close()

	key, err := api.LoadPrivateKeyFromFile(privatePath)
	if err != nil {
		panic(err)
	}
	_ = key

	interruptSignal := make(chan os.Signal, 1)
	signal.Notify(interruptSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	fmt.Printf("Starting %d concurrent loops. Press Ctrl+C to stop...\n", concurrency)

	stopCh := make(chan struct{})
	go func() {
		<-interruptSignal
		fmt.Println("Stopping...")
		close(stopCh)
	}()

	var wg sync.WaitGroup
	var totalIterations atomic.Int64
	var lastPrinted atomic.Int64
	if verbose {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
				}
				current := totalIterations.Load()
				if current >= lastPrinted.Load()+int64(100*concurrency) {
					fmt.Printf("total iterations: %d\n", current)
					lastPrinted.Store(current)
				}
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}
	for workerID := 0; workerID < concurrency; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			iteration := int64(0)
			for {
				select {
				case <-stopCh:
					return
				default:
				}

				iteration++
				totalIterations.Add(1)
				conn, err := db.Conn(ctx)
				if err != nil {
					if verbose {
						fmt.Printf("[worker %d] connect failed at iteration %d: %v\n", workerID, iteration, err)
					}
					time.Sleep(100 * time.Millisecond)
					continue
				}

				row := conn.QueryRowContext(ctx, fmt.Sprintf("select %d", iteration))
				if row.Err() != nil {
					if verbose {
						fmt.Printf("[worker %d] query failed at iteration %d: %v\n", workerID, iteration, row.Err())
					}
					conn.Close()
					time.Sleep(100 * time.Millisecond)
					continue
				}

				var v int64
				if err := row.Scan(&v); err != nil {
					if verbose {
						fmt.Printf("[worker %d] scan failed at iteration %d: %v\n", workerID, iteration, err)
					}
					conn.Close()
					time.Sleep(100 * time.Millisecond)
					continue
				}

				if v != iteration {
					fmt.Printf("[worker %d] unexpected query result at iteration %d: %d\n", workerID, iteration, v)
				}

				row = conn.QueryRowContext(ctx, "select value from v$sysstat where name = ?", "APPEND_DATA_SUCCESS")
				if row.Err() != nil {
					if verbose {
						fmt.Printf("[worker %d] query failed at iteration %d: %v\n", workerID, iteration, row.Err())
					}
					conn.Close()
					time.Sleep(100 * time.Millisecond)
					continue
				}

				var appendDataSuccess int64
				if err := row.Scan(&appendDataSuccess); err != nil {
					if verbose {
						fmt.Printf("[worker %d] scan failed at iteration %d: %v\n", workerID, iteration, err)
					}
					conn.Close()
					time.Sleep(100 * time.Millisecond)
					continue
				}

				conn.Close()
				time.Sleep(10 * time.Millisecond)
			}
		}(workerID)
	}

	wg.Wait()
	fmt.Printf("completed %d total iterations\n", totalIterations.Load())
}

func ensureAuthKeyPair(dir string, prefix string) (string, string, error) {
	privatePath := filepath.Join(dir, prefix+"_private.pem")
	publicPath := filepath.Join(dir, prefix+"_public.pem")

	if _, err := os.Stat(privatePath); err == nil {
		if _, err := os.Stat(publicPath); err == nil {
			pubBytes, err := os.ReadFile(publicPath)
			if err != nil {
				return "", "", err
			}
			return privatePath, string(pubBytes), nil
		}
	}

	pair, err := api.GenerateAuthKeyPair()
	if err != nil {
		return "", "", err
	}

	privatePath, publicPath, err = pair.WriteFiles(dir, prefix)
	if err != nil {
		return "", "", err
	}

	pubBytes, err := os.ReadFile(publicPath)
	if err != nil {
		return "", "", err
	}
	return privatePath, string(pubBytes), nil
}

func registerAuthKey(ctx context.Context, conn *sql.Conn, user string, publicPEM string) error {
	publicPEM = strings.TrimSpace(publicPEM)

	row := conn.QueryRowContext(ctx, `SELECT KEY_ID, ACTIVATED FROM V$USER_AUTH_KEYS WHERE USER_NAME=? AND PUBKEY=? ORDER BY KEY_ID DESC LIMIT 1`, user, publicPEM)
	if row.Err() == nil {
		var keyID int
		var activated int
		if err := row.Scan(&keyID, &activated); err == nil {
			if activated == 0 {
				_, err := conn.ExecContext(ctx, fmt.Sprintf("ALTER USER %s ACTIVATE AUTH KEY ID %d", user, keyID))
				if err != nil {
					return err
				}
			}
			fmt.Printf("Auth key already registered for user %s (key_id=%d)\n", user, keyID)
			return nil
		}
	}

	validBefore := time.Now().Add(24 * time.Hour * 365 * 30).Format("2006-01-02")
	comment := fmt.Sprintf("authaging %d", time.Now().UnixNano())

	_, err := conn.ExecContext(ctx, fmt.Sprintf("ALTER USER %s ADD AUTH KEY (KEY = '%s', VALID_BEFORE = '%s', COMMENT = '%s')", user, publicPEM, validBefore, comment))
	if err != nil {
		return err
	}

	row = conn.QueryRowContext(ctx, `SELECT KEY_ID, ACTIVATED FROM V$USER_AUTH_KEYS WHERE USER_NAME=? AND COMMENT=? ORDER BY KEY_ID DESC LIMIT 1`, user, comment)
	if row.Err() != nil {
		return row.Err()
	}

	var keyID int
	var activated int
	if err := row.Scan(&keyID, &activated); err != nil {
		return err
	}

	if activated == 0 {
		_, err := conn.ExecContext(ctx, fmt.Sprintf("ALTER USER %s ACTIVATE AUTH KEY ID %d", user, keyID))
		if err != nil {
			return err
		}
	}

	fmt.Printf("Registered auth key for user %s (key_id=%d)\n", user, keyID)
	return nil
}
