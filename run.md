# Init
go mod init mananger_pcs

# Running
go run generate_keys.go
go run agente_http.go
go run servidor_http.go

# Compiling Go Files into Executables
go build generate_keys.go
go build agente_http.go
go build servidor_http.go

# Creating Optimized Executables
go build -ldflags="-s -w" generate_keys.go
go build -ldflags="-s -w" agente_http.go
go build -ldflags="-s -w" servidor_http.go

# Cross-Compiling (Optional)
set GOOS=linux
set GOARCH=amd64
go build generate_keys.go

# Redes
10.46 - Henoch
10.45 - Euza