# Fases de Melhoria da Linguagem ION

Este arquivo rastreia todas as melhorias propostas, organizadas por fase. Cada fase é independente e pode ser implementada em qualquer ordem.

**Legenda:**
- 📌 Pendente
- 🔄 Em andamento
- ✅ Concluído

---

## Fase 1: Correção do Bug do Brainfuck (Escopo vs Bloco)

| # | Tarefa | Status | Arquivos Afetados | Descrição |
|---|--------|--------|-------------------|-----------|
| 1.1 | Implementar escopo de bloco real | ✅ | `compiler/compiler.go` | Substituir o escopo de função atual por escopo de bloco aninhado. O interpretador Brainfuck usa `declare` dentro de `if` e espera que essas variáveis sejam locais ao bloco, não içadas para o topo da função |
| 1.2 | Adicionar SymbolTable aninhada (Outer) | ✅ | `compiler/compiler.go` | Criar linked list de SymbolTables (Outer *SymbolTable) para resolução de escopo em cascata |
| 1.3 | Atualizar compileDeclareStatement | ✅ | `compiler/compiler.go` | Ao entrar em um bloco (if, while, for), criar novo escopo. Ao sair, remover locais do escopo |
| 1.4 | Atualizar compileIdentifier para resolução em cascata | ✅ | `compiler/compiler.go` | Resolver variáveis procurando no escopo atual, depois no outer, até chegar no global |
| 1.5 | Testar com brainfuck.ion | ✅ | `examples/BrainFuck.ion` | Verificar que o interpretador Brainfuck roda sem erro. Declares agora funcionam dentro de blocos |
| 1.6 | Adicionar testes de regressão de escopo | ✅ | `examples/` | Criar casos de teste para shadowing, variáveis em if aninhados, while aninhados |

---

## Fase 2: Lexer - Melhorias na Tokenização

| # | Tarefa | Status | Arquivos Afetados | Descrição |
|---|--------|--------|-------------------|-----------|
| 2.1 | Suporte a números de ponto flutuante | ✅ | `lexer/lexer.go` | Aceitar `10.5`, `.5`, `3.14`, notação científica `1.5e10` |
| 2.2 | Escape sequences em strings | ✅ | `lexer/lexer.go` | Suporte a `\n`, `\"`, `\\`, `\t`, `\r` em readString() |
| 2.3 | Underscore em números | ✅ | `lexer/lexer.go` | Aceitar `_` como separador visual em números: `1_000_000` |
| 2.4 | Comentários de bloco `/* ... */` | ✅ | `lexer/lexer.go` | Adicionar suporte a comentários multilinha em skipWhitespace() |
| 2.5 | Números hexadecimais/binary/octal | ✅ | `lexer/lexer.go` | Suporte a `0xFF`, `0b1010`, `0o77` |
| 2.6 | Corrigir coluna off-by-one no lexer | ✅ | `lexer/lexer.go` | `readChar()` agora mantém `column` = coluna de `l.ch` (1-indexed); removido `-1` de todas as criações de token |
| 2.7 | String não terminada retorna `\x00` | ✅ | `lexer/lexer.go` | Em vez de literal `"STRING NÃO TERMINADA"`, retorna `\x00` que o parser trata como string vazia |
| 2.8 | LookupIdent case-insensitive | ✅ | `lexer/token.go` | Adicionado `toLower()` para que `IF`, `While`, etc. sejam reconhecidos como keywords |
| 2.9 | **Dígitos em identificadores** | ✅ | `lexer/lexer.go` | `readIdentifier()` agora aceita `isLetter()` OU `isDigit()`, permitindo nomes como `s1`, `var2` |

---

## Fase 3: Compilador - Type Checker e Validação

| # | Tarefa | Status | Arquivos Afetados | Descrição |
|---|--------|--------|-------------------|-----------|
| 3.1 | Verificação de tipos em tempo de compilação | ✅ | `compiler/compiler.go` | Type stack com pushType/popType/peekType. Verificação de operandos em +, -, *, /, >, <, ==, !=. Built-ins (char, len, get_byte_at, input) validam tipos dos argumentos |
| 3.2 | Tipos de parâmetros de função | ✅ | `parser/ast.go`, `parser/parser.go`, `compiler/compiler.go` | Criado tipo `Parameter` com `Name` e `Type`. Parser aceita `param: type`. Compiler usa `getTypeFromTypeNode` para type check |
| 3.3 | Verificação de tipo em declarações com init | ✅ | `compiler/compiler.go` | `declare x: number := "texto"` gera erro de compilação. Nil é atribuível a qualquer tipo |
| 3.4 | Inferência de tipo em atribuição | ✅ | `compiler/compiler.go` | `:=` verifica se o tipo do valor corresponde ao tipo declarado da variável |
| 3.5 | Verificação de tipo em array | ✅ | `compiler/compiler.go` | `compileIndexExpression` verifica se o alvo é um array |
| 3.6 | compileOrdExpression agora usa argumento | ✅ | `compiler/compiler.go`, `parser/ast.go`, `parser/parser.go`, `vm/vm.go` | `ord()` compila o argumento (como `char()`), popa da pilha e retorna o byte do primeiro caractere |
| 3.7 | Limite de índices >255 com erro | ✅ | `compiler/compiler.go` | Adicionado `if symbol.Index > 255` em todos os 6 locais que usavam `byte(symbol.Index)` |
| 3.8 | For com step negativo | ✅ | `compiler/compiler.go` | Detecta step negativo via NumberLiteral e usa OP_LESS em vez de OP_GREATER |
| 3.9 | Variáveis globais | ✅ | `compiler/compiler.go` | `scopeDepth++` removido do `Compile()`. Blocos (if/while/for) agora incrementam `scopeDepth` para que declares dentro de blocos sejam locais. Declares no script principal viram GLOBAIS |

---

## Fase 4: VM - Expansão de Limites e Performance

| # | Tarefa | Status | Arquivos Afetados | Descrição |
|---|--------|--------|-------------------|-----------|
| 4.1 | Expandir constantes para 2-byte index | 📌 | `vm/chunk.go`, `vm/vm.go`, `compiler/compiler.go` | Pendente: adicionar OP_CONSTANT_16 para >256 constantes. Atualmente erro em 255 |
| 4.2 | Expandir globals para 2-byte index | 📌 | `vm/vm.go`, `compiler/compiler.go` | Pendente: adicionar OP_DEFINE_GLOBAL_16 etc. Atualmente limite de 256 |
| 4.3 | Expandir locals para 2-byte index | 📌 | `vm/vm.go`, `compiler/compiler.go` | Pendente: adicionar OP_GET_LOCAL_16 etc. Atualmente checka >255 |
| 4.4 | STACK_MAX dinâmico e configurável | ✅ | `vm/vm.go` | `NewVMConfig(stackMax, framesMax)`. Padrão: 4096 stack, 256 frames |
| 4.5 | Otimização de constantes (dedup) | ✅ | `vm/chunk.go` | `AddConstant` agora compara `Type` e `As` antes de adicionar |
| 4.6 | Run-Length Encoding para line mapping | 📌 | `vm/chunk.go` | Pendente: substituir array 1:1 de Lines por RLE |
| 4.7 | STACK_MAX e FRAMES_MAX configuráveis | ✅ | `vm/vm.go` | `NewVMConfig()` aceita stackMax e framesMax. Constantes `DefaultStackMax=4096`, `DefaultFramesMax=256` |
| 4.8 | Panics na VM tratados como erros | ✅ | `vm/vm.go` | `Run()` com `defer recover()` captura panics e retorna erro. Limites usam `len(vm.stack)`/`len(vm.frames)` |
| 4.9 | Bounds checks em índices de locais | ✅ | `vm/vm.go` | OP_GET_LOCAL e OP_SET_LOCAL validam `absSlot` contra `len(vm.stack)`. OP_CALL valida `stackSlot >= 0` |
| 4.10 | runtimeError com ip=0 não crasha | ✅ | `vm/vm.go` | `if frame.ip > 0` antes de `GetLine(frame.ip - 1)` |
| 4.11 | **isTruthy** | 📌 | `vm/vm.go` | Pendente: type checking para if/while (compilador já faz). `isTruthy` mantém comportamento existente |

---

## Fase 5: Novas Features da Linguagem

| # | Tarefa | Status | Arquivos Afetados | Descrição |
|---|--------|--------|-------------------|-----------|
| 5.1 | Operador `<=` e `>=` | ✅ | `lexer`, `parser`, `compiler`, `vm/opcode.go`, `vm/vm.go` | Adicionado tokens `TOKEN_GREATER_EQUAL`/`TOKEN_LESS_EQUAL`, lexer com `peekChar()=='='`, parser com precedência `LESSGREATER`, compilador emite `OP_GREATER_EQUAL`/`OP_LESS_EQUAL` |
| 5.2 | Operador `%` (módulo) | ✅ | `lexer`, `parser`, `compiler`, `vm` | Adicionado `TOKEN_PERCENT`, precedência `PRODUCT`, compilador emite `OP_MODULO` |
| 5.3 | Operadores compostos `+=`, `-=`, `*=`, `/=` | ✅ | `lexer`, `parser`, `compiler` | Tokens `TOKEN_PLUS_ASSIGN` etc., parser com mapa `assignTokens`, compilador com `emitCompoundOp()` — GET+OP+SET |
| 5.4 | Else if | ✅ | `parser/parser.go` | Já funcionava: `else` → `parseBlockStatement` → `parseStatement` para `if` aninhado |
| 5.5 | Break e Continue em loops | ✅ | `parser`, `compiler` | Tokens `TOKEN_BREAK`/`TOKEN_CONTINUE`, AST `BreakStatement`/`ContinueStatement`, compiler com `loopStack`/`breakStack` |
| 5.6 | REPL (Read-Eval-Print Loop) | 📌 | Novo: `ionr/` | Pendente |
| 5.7 | Sistema de módulos/import | 📌 | `parser`, `compiler` | Pendente |
| 5.8 | Step opcional no for | ✅ | `parser/parser.go` | `step` opcional: se `peekToken` não for `step`, usa `NumberLiteral{Value:1}` como padrão |
| 5.9 | **String + number concatenação** | 📌 | `compiler/compiler.go`, `vm/vm.go` | Pendente: `OP_ADD` rejeita strings |
| 5.10 | **Maps/structs/lists** | 📌 | Múltiplos arquivos | Pendente |

---

## Fase 6: Funções e Built-ins

| # | Tarefa | Status | Arquivos Afetados | Descrição |
|---|--------|--------|-------------------|-----------|
| 6.1 | Funções recursivas | ✅ | `vm/vm.go` | Testado com fatorial recursivo. CallFrame system funciona (`fatorial(5) = 120`) |
| 6.2 | Closures | 📌 | `compiler/compiler.go`, `vm/vm.go` | Pendente |
| 6.3 | Funções nativas em Go | 📌 | `vm/vm.go` | Pendente: sistema de registro |
| 6.4 | `tostring()`, `tonumber()` | ✅ | `vm/vm.go`, `compiler/compiler.go`, `parser` | `tostring(42)` → `"42"`. `tonumber("3.14")` → `3.14`. Inválido retorna `nil` |
| 6.5 | `readfile()` / `writefile()` | ✅ | `vm/vm.go`, `compiler/compiler.go`, `parser` | `readfile("path")` → string ou nil. `writefile("path","content")` → escreve arquivo |
| 6.6 | `exit(code)` | ✅ | `vm/vm.go`, `compiler/compiler.go` | `OP_EXIT` executa `os.Exit(code)` |
| 6.7 | Tipo de retorno em funções | 📌 | `parser/ast.go`, `compiler/compiler.go` | Pendente |

---

## Fase 7: Mensagens de Erro e Debug

| # | Tarefa | Status | Arquivos Afetados | Descrição |
|---|--------|--------|-------------------|-----------|
| 7.1 | Mensagens de erro mais descritivas (lexer) | 📌 | `lexer/lexer.go` | Incluir contexto (linha, coluna, snippet) |
| 7.2 | Mensagens de erro mais descritivas (parser) | 📌 | `parser/parser.go` | Sugerir correções. Erro inconsistente: parseIfStatement retorna nil (perde AST parcial), outros retornam parcial |
| 7.3 | Mensagens de erro mais descritivas (compiler) | 📌 | `compiler/compiler.go` | "variável 'x' não declarada" com sugestão de declare; incluir coluna (não apenas linha) |
| 7.4 | Stack trace em erros de runtime | 📌 | `vm/vm.go` | Mostrar pilha de chamadas no erro |
| 7.5 | Modo debug na VM (trace de opcodes) | 📌 | `vm/vm.go` | Ativar/desativar dump de execução com flag |
| 7.6 | Disassembler / dump de bytecode | 📌 | Novo: `iondis` | Ferramenta para inspecionar bytecode .ionc |
| 7.7 | **emitReturn usa linha fixa 1** | 📌 | `compiler/compiler.go:570` | `line := 1 // TODO: Rastrear linha` - erros em retorno implícito mostram linha errada |

---

## Fase 8: Limpeza e Refatoração

| # | Tarefa | Status | Arquivos Afetados | Descrição |
|---|--------|--------|-------------------|-----------|
| 8.1 | Remover código comentado e dead code | 📌 | Todos | `defineGlobal()` (nunca chamado), `defineLocal()` (nunca chamado), `ArrayTypeLiteral`, `ExpressionStatement`, `parseExpressionStatement`, comentários de versão obsoletos (V13, V13.6) |
| 8.2 | Padronizar nomes de funções/variáveis | 📌 | Todos | Mistura de português e inglês em mensagens de erro, comentários, nomes de variáveis |
| 8.3 | Extrair magic numbers para constantes | 📌 | `compiler/compiler.go`, `vm/vm.go` | Substituir 255, 256, 65535 por constantes nomeadas |
| 8.4 | Tratar panics na VM como erros | 📌 | `vm/vm.go` | Substituir push/pop/peek com `panic` por tratamento de erro com `(Value, error)` |
| 8.5 | Remover dependência `ioutil` (deprecada) | 📌 | `ionc/main.go` | Substituir `ioutil.ReadFile` por `os.ReadFile` |
| 8.6 | Testes unitários automatizados | 📌 | Novo: `*_test.go` | Criar testes Go para lexer, parser, compiler, VM (zero cobertura atualmente) |
| 8.7 | CI/CD pipeline | 📌 | `.github/` | GitHub Actions para build e testes automáticos |
| 8.8 | **patchJump escreve mesmo com erro** | 📌 | `compiler/compiler.go:921-929` | Quando salto >65535, adiciona erro mas continua escrevendo bytes corrompidos no chunk |
| 8.9 | **scopeDepth usado como booleano** | 📌 | `compiler/compiler.go:61` | `scopeDepth` é incrementado só para o script (0→1), nunca para blocos. Nome enganoso, deveria ser `isFunctionScope` |
| 8.10 | **isLocal reusado com 2 significados** | 📌 | `compiler/compiler.go:277,658` | Na linha 277 é "está em função?", na linha 658 é "encontrou na symbol table?" |

---

## Resumo de Progresso

| Fase | Total | ✅ Concluído | 📌 Pendente | Progresso |
|------|-------|--------------|-------------|-----------|
| 1 - Bug Brainfuck | 6 | 6 | 0 | **100%** |
| 2 - Lexer | 9 | 9 | 0 | **100%** |
| 3 - Type Checker | 9 | 9 | 0 | **100%** |
| 4 - VM Limites | 11 | 7 | 4 | **64%** |
| 5 - Novas Features | 10 | 6 | 4 | **60%** |
| 6 - Funções | 7 | 4 | 3 | **57%** |
| 7 - Erros/Debug | 7 | 0 | 7 | **0%** |
| 8 - Refatoração | 10 | 0 | 10 | **0%** |
| **Total** | **69** | **41** | **28** | **59%** |
