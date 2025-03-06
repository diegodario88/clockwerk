# Clockwerk ⏰

**Clockwerk** é uma aplicação TUI (Terminal User Interface) escrita em Go para gestão eletrônica de ponto de trabalho [Senior X](https://www.senior.com.br). Gerencie seus registros de jornada diretamente do terminal, substituindo aplicativos tradicionais por uma interface simples e eficiente.

![Demonstração do Clockwerk](https://github.com/user-attachments/assets/376f75f6-4e8b-49ab-8908-1c795df61543)

## ✨ Funcionalidades

- **Registro de entrada/saída**
  - Inicie e encerre sua jornada com comandos intuitivos
- **Gestão de intervalos**
  - Controle pausas para almoço e descanso
- **Interface amigável**
  - Navegação simplificada via teclado
  - Visualização em tempo real dos registros
- **Multiplataforma**
  - Compatível com Windows, Linux e macOS
- **Leve e rápido**
  - Consumo mínimo de recursos (CPU/RAM)

### Navegação

As seguinte teclas de atalho são usadas para navegar e explorar o programa

- <kbd>h</kbd>/<kbd>←</kbd> para navegar para esquerda
- <kbd>l</kbd>/<kbd>→</kbd> para navegar para direita
- <kbd>Space</kbd> bater o ponto
- <kbd>ctrl+c</kbd> forçar fechar o programa
- <kbd>q</kbd> sair
- <kbd>e</kbd> esquecer credenciais
- <kbd>r</kbd> retentar caso haja erro

## 📥 Instalação

### Binários Pré-Compilados

1. Acesse a [página de releases](https://github.com/diegodario88/clockwerk/releases)
2. Baixe o executável para seu sistema:
   - Windows: `clockwerk_windows_amd64.exe`
   - Linux: `clockwerk_linux_amd64`
   - macOS: `clockwerk_darwin_amd64`

### Via Código Fonte

Requisitos:

- Go 1.22+
- `mingw-w64` (para compilação no Windows)

```bash
# Instalar e executar
go install github.com/diegodario88/clockwerk@latest

# Compilar manualmente (Windows)
GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ CGO_ENABLED=1 go build -o clockwerk.exe
```
