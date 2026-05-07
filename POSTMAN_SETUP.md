# Como usar Postman para testar DocuSeal

## 1. Importar no Postman

1. Abra o Postman
2. Clique em **Import** (canto superior esquerdo)
3. Selecione o arquivo `Postman_DocuSeal_Test.json`
4. Clique em **Import**

## 2. Configurar variáveis

O arquivo já vem com `base_url` = `http://localhost:8080` (padrão local).

Se precisar mudar:
1. No Postman, vá em **Environments** (ou clique no olho no canto superior direito)
2. Edite a variável `base_url` para o seu servidor
3. Selecione esse environment na lista

## 3. Testar

1. Abra a request **"DocuSeal - Gerar Documento"**
2. Na aba **Body**, edite os campos com seus dados de teste:
   - `email`: seu email para receber o PDF assinado
   - Outros campos: dados fictícios para preencher o documento
3. Clique em **Send**

## 4. Resposta esperada

```json
{
  "signing_url": "https://app.docuseal.com/submissions/xxxxx/request_signature/yyyy",
  "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

## 5. Próximos passos

1. Copie a `signing_url`
2. Abra em um navegador para assinar o documento
3. Depois de assinar, o webhook DocuSeal notificará seu servidor
4. Um email será enviado com o PDF assinado

---

**Pronto! Agora é só usar o Postman normalmente para testar.** 🚀
