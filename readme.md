# Sistema de Gerenciamento de Computadores

Este projeto é composto por vários componentes que trabalham em conjunto para monitorar e gerenciar computadores em uma rede. Abaixo está a descrição detalhada de cada componente:

## Agente HTTP (agente_http)

O Agente HTTP é um programa cliente que deve ser instalado em cada computador que se deseja monitorar. Suas principais características são:

- Porta padrão: 9999
- Coleta automática de informações do sistema:
  - Hardware (CPU, Memória, Discos)
  - Sistema Operacional
  - Rede
  - Processos em execução
- Auto-atualização automática
- Inicialização automática com o Windows
- Banco de dados SQLite local
- Intervalo configurável para coleta de informações
- Criptografia de dados usando chaves públicas/privadas

## Servidor HTTP (servidor_http)

O Servidor HTTP é responsável por descobrir e coletar informações dos agentes na rede. Suas principais funcionalidades são:

- Descoberta automática de agentes na rede
- Monitoramento periódico (padrão: 30 minutos)
- Suporte a múltiplas redes
- Sistema de workers para consultas paralelas
- Armazenamento de informações em banco de dados
- Processamento de dados criptografados
- Exibição de estatísticas de computadores monitorados

## Servidor de Atualização (servidor_atualizacao)

O Servidor de Atualização é responsável por distribuir novas versões do agente. Características principais:

- Porta padrão: 9991
- Distribuição de atualizações do agente
- Gerenciamento de chaves públicas/privadas
- Estatísticas de downloads e clientes
- Timeouts configuráveis
- Suporte a arquivos estáticos
- Monitoramento de clientes ativos

## Commander (commander)

O Commander é uma ferramenta de linha de comando para interagir com os agentes. Principais funcionalidades:

- Consulta de informações específicas dos agentes:
  - CPU
  - Discos
  - GPU
  - Hardware
  - Memória
  - Processos
  - Rede
  - Sistema
  - Informações do agente
- Atualização do IP do servidor de atualização
- Configuração de intervalos de atualização
- Suporte a timeout configurável

## Requisitos do Sistema

- Sistema Operacional Windows (para algumas funcionalidades específicas do agente)
- Acesso à rede local
- Permissões de administrador para instalação do agente

## Segurança

O sistema utiliza:
- Criptografia de dados usando chaves públicas/privadas
- Autenticação entre componentes
- Proteção contra acessos não autorizados
- Validação de integridade das atualizações

## Configuração

1. Gerar chaves públicas/privadas usando o script `generate_keys.exe`
2. Distribuir a chave pública para os agentes. A chave 'public_key.pem' deve ser copiada para o diretório 'keys' do agente.
3. Configurar o servidor de atualização com a chave privada
4. Instalar e configurar o agente nos computadores alvos
5. Iniciar o servidor HTTP para monitoramento

## Portas Utilizadas

- Agente HTTP: 9999
- Servidor de Atualização: 9991

## Observações

- O sistema é projetado para ambientes Windows
- Ao executar o agente, ele se configura automaticamente a parta do firewall para permitir comunicação com o servidor HTTP
- Os intervalos de atualização podem ser ajustados conforme necessidade

        