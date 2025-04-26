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


# Commander
commander.exe -agent 10.46.102.245:9999 -update-ip http://10.46.102.245:9991
commander.exe -agent 10.46.102.245 -info
commander.exe -agent 10.46.102.245 -sys-interval 15
commander.exe -agent 10.46.102.245 -update-interval 30

commander.exe -agent all -update-ip http://10.46.102.245:9991
commander.exe -agent all -sys-interval 15
commander.exe -agent all -update-interval 30

commander.exe -agent 192.168.1.4 -cmd "ping 8.8.8.8"
commander.exe -agent 192.168.1.5 -cmd "msg * /time:60 Mensagem de teste"
commander.exe -agent 192.168.1.4 -cmd "dir /x c:\" # Mostrar os arquivos e diretórios da raiz do C:" com nomes curtos
commander.exe -agent 192.168.1.4 -cmd "dir C:\LIVROS~1" # Livro é uma pasta que tem espaço no nome.. 'Livros Estudar'
    - dir /X mostrar os nomes curtos dos arquivos e diretórios.
    - dir /S mostrar os nomes completos dos arquivos e diretórios.
    - dir /B mostrar apenas os nomes dos arquivos e diretórios.
    - dir /A mostrar apenas os arquivos e diretórios ocultos.
    - type funciona com nome curtos: NOVO6~1.TXT ('novo 6.txt')

commander.exe -agent 192.168.1.4 -ps "Get-Process | Select-Object -First 5"
commander.exe -agent 192.168.1.4 -ps "Get-Process | Where-Object {$_.ProcessName -like \"*agente*\"} | Select-Object Name, Id, Path"
