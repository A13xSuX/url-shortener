package main

import (
    "database/sql"
    "fmt"
    "log"
    "time"
    
    _ "github.com/lib/pq"
)

func main() {
    fmt.Println("Тестирование из WSL...")
    
    testDSNs := []string{
        // Localhost изнутри WSL
        "host=localhost port=5432 user=postgres password=postgres dbname=mydb sslmode=disable",
        "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=mydb sslmode=disable",
        
        // WSL IP
        "host=172.30.109.15 port=5432 user=postgres password=postgres dbname=mydb sslmode=disable",
        
        // Имя контейнера (если в одной сети)
        "host=postgres port=5432 user=postgres password=postgres dbname=mydb sslmode=disable",
    }
    
    for i, dsn := range testDSNs {
        fmt.Printf("\nТест %d: %s\n", i+1, dsn)
        
        db, err := sql.Open("postgres", dsn)
        if err != nil {
            log.Printf("❌ Ошибка Open: %v", err)
            continue
        }
        
        db.SetConnMaxLifetime(2 * time.Second)
        
        err = db.Ping()
        if err != nil {
            log.Printf("❌ Ошибка Ping: %v", err)
        } else {
            fmt.Printf("✅ УСПЕХ!\n")
            
            var version string
            err = db.QueryRow("SELECT version()").Scan(&version)
            if err == nil {
                fmt.Printf("PostgreSQL: %s\n", version)
            }
            
            db.Close()
            return
        }
        
        db.Close()
    }
    
    fmt.Println("\n⚠️  Ни один вариант не сработал из WSL.")
    fmt.Println("Проверьте, запущен ли PostgreSQL:")
    fmt.Println("docker-compose ps")
    fmt.Println("docker-compose logs postgres")
}
