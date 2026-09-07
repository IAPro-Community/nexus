# Missions e Autopilot

Mission é a camada avançada para executar trabalho aprovado em etapas coordenadas:

```text
READY → ALLOCATE → COMPILE_PROMPT → EXECUTE → TEST → REVIEW → VERIFY
```

O runtime deve concluir por evidência, não porque um Agent disse “done”. Use Mission quando o trabalho justificar duração, dependências, múltiplos workers, gates ou recuperação. Não é requisito para uma AI Session direta.
