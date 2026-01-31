# 💎 Ion Language v1.13

Ion é uma linguagem de programação didática, com tipagem estática e execução baseada em **Bytecode** através de uma Máquina Virtual (VM) customizada, desenvolvida inteiramente em Go.



## 🚀 Funcionalidades Atuais
- **Tipos de Dados:** `number` (float64), `string` e `bool`.
- **Variáveis:** Declaração explícita com `declare` e inferência/atribuição com `:=`.
- **Estruturas de Controle:** - Loops `for ... to ... step ... next`.
  - Condicionais `if ... then ... else ... endif`.
- **Operadores:**
  - Matemáticos: `+`, `-`, `*`, `/`.
  - Comparação: `==`, `>`, `<`.
- **Saída:** Comando `display` para impressão no console.

## 🛠️ Como Gerar os Executáveis (Build)

Se você acabou de baixar o código-fonte, precisa gerar o compilador e a VM:

```bash
# Gerar o Compilador (ionc.exe)
go build -o ionc.exe ./ionc

# Gerar a Máquina Virtual (ionv.exe)
go build -o ionv.exe ./ionv

💻 Como Usar a Linguagem
O fluxo de execução do Ion segue duas etapas obrigatórias:

1. Compilação para Bytecode
O compilador transforma seu texto (.ion) em instruções que a VM entende (.ionc).

Bash
./ionc.exe -s examples/math_test_v3.ion -o math_test.ionc

2. Execução na VM
A VM lê o arquivo binário gerado e executa o programa.

Bash
./ionv.exe math_test.ionc

📂 Estrutura do Projeto
/ionc: Código-fonte do compilador (Lexer, Parser, AST, Compiler).
/ionv: Código-fonte do motor da Máquina Virtual.
/pkg: Pacotes compartilhados de lógica da linguagem.
/examples: Arquivos de teste .ion para você explorar a sintaxe.


📝 Exemplo de Código
Snippet de código
begin program
    declare x: number := 10
    if x > 5 then
        display "Olá do Ion! x é maior que 5."
    endif
end program

---

### 🛠️ Facilitando sua vida: `build.bat`

Para você não ter que ficar digitando os comandos de `go build` toda hora, crie um arquivo na raiz chamado `build.bat` com este conteúdo:

```batch
@echo off
echo Gerando binarios do Ion...
go build -o ionc.exe ./ionc
go build -o ionv.exe ./ionv
echo Concluido! ionc.exe e ionv.exe criados.

