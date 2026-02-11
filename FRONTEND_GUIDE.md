# 🎨 Guia de Implementação Frontend - Dependentes

## 📍 Endpoints Disponíveis

### 1. **POST /checkout** - Criar Cliente e Checkout
```typescript
// Request
interface CheckoutRequest {
  // Dados do titular
  name: string;
  email: string;
  cpf: string;
  phone: string;
  birth_date: string; // "YYYY-MM-DD"
  gender: string; // "1", "2", "3"
  plan_id: string;
  payment_method: "PIX" | "CREDIT_CARD";
  
  // Endereço
  street: string;
  number: string;
  complement?: string;
  district: string;
  city: string;
  state: string;
  zip_code: string;
  
  // Cartão (se payment_method === "CREDIT_CARD")
  card_holder?: string;
  card_number?: string;
  card_month?: string;
  card_year?: string;
  card_cvv?: string;
  
  // Termos
  terms_accepted: boolean;
  terms_accepted_at: string; // ISO 8601
  terms_version: string;
  
  // 🆕 DEPENDENTES (OPCIONAL)
  dependents?: Array<{
    name: string;
    cpf: string;
    birth_date: string; // "YYYY-MM-DD"
    gender: string; // "1", "2", "3"
    kinship: string; // "FILHO", "CONJUGE", "PAI", "MAE"
  }>;
}

// Response
interface CheckoutResponse {
  id: string;           // ID do customer
  name: string;
  email: string;
  status: string;       // "PENDING" ou "ACTIVE"
  msg: string;
  pix_code: string;     // Copia e cola (vazio se cartão)
  pix_qr_code_url: string; // Base64 PNG (vazio se cartão)
}
```

### 2. **GET /customers/:id/status** - Verificar Status do Pagamento
```typescript
// Response
{
  status: "PENDING" | "ACTIVE" | "SUSPENDED" | "CANCELLED"
}
```

---

## 🎯 Fluxos de UX Recomendados

### **Opção 1: Toggle "Tem Dependentes?" (Mais Simples)**

```tsx
// Step no formulário de checkout
<FormSection>
  <h3>Titular do Plano</h3>
  {/* Campos do titular */}
  
  <Divider />
  
  <SwitchField 
    label="Deseja adicionar dependentes?"
    onChange={(checked) => setHasDependents(checked)}
  />
  
  {hasDependents && (
    <DependentsSection>
      <p className="text-sm text-gray-600">
        Adicione cônjuge, filhos ou outros familiares ao plano.
      </p>
      
      {dependents.map((dep, index) => (
        <DependentCard key={index}>
          <Input label="Nome Completo" {...} />
          <Input label="CPF" mask="000.000.000-00" {...} />
          <DateInput label="Data de Nascimento" {...} />
          <Select label="Gênero">
            <option value="1">Masculino</option>
            <option value="2">Feminino</option>
            <option value="3">Outro</option>
          </Select>
          <Select label="Parentesco">
            <option value="CONJUGE">Cônjuge</option>
            <option value="FILHO">Filho(a)</option>
            <option value="PAI">Pai</option>
            <option value="MAE">Mãe</option>
            <option value="IRMAO">Irmão(ã)</option>
          </Select>
          <Button onClick={() => removeDep(index)}>Remover</Button>
        </DependentCard>
      ))}
      
      <Button onClick={addDependent}>
        + Adicionar Dependente
      </Button>
    </DependentsSection>
  )}
</FormSection>
```

**✅ Vantagens:**
- Simples e direto
- Não sobrecarrega quem não tem dependentes
- Menos passos no fluxo

---

### **Opção 2: Step Dedicado no Multi-Step Form**

```tsx
// Step 1: Dados Pessoais
// Step 2: Endereço
// Step 3: 🆕 Dependentes (Opcional - pode pular)
// Step 4: Pagamento

<StepDependents>
  <h2>Adicionar Dependentes (Opcional)</h2>
  <p>Você pode adicionar dependentes agora ou depois pelo painel.</p>
  
  {dependents.length === 0 ? (
    <EmptyState>
      <Icon name="users" />
      <p>Nenhum dependente adicionado</p>
      <Button onClick={addFirst}>Adicionar Primeiro Dependente</Button>
      <Button variant="ghost" onClick={skipStep}>Pular Esta Etapa</Button>
    </EmptyState>
  ) : (
    <>
      {dependents.map((dep) => <DependentCard {...dep} />)}
      <Button onClick={addAnother}>+ Adicionar Outro</Button>
      <Button onClick={nextStep}>Continuar para Pagamento</Button>
    </>
  )}
</StepDependents>
```

**✅ Vantagens:**
- Fluxo mais organizado
- Não mistura dados do titular com dependentes
- Permite explicar melhor a feature

---

### **Opção 3: Accordion/Collapsible (Mais Compacto)**

```tsx
<Accordion defaultExpanded={false}>
  <AccordionSummary>
    <div className="flex justify-between w-full">
      <span>Dependentes (Opcional)</span>
      <Badge>{dependents.length} adicionados</Badge>
    </div>
  </AccordionSummary>
  
  <AccordionDetails>
    {/* Formulário de dependentes */}
  </AccordionDetails>
</Accordion>
```

**✅ Vantagens:**
- Economiza espaço vertical
- Boa para formulários longos
- Mostra contador de dependentes

---

## 🖼️ Tela de PIX QR Code

### ⚠️ **SITUAÇÃO ATUAL:** Não existe rota dedicada para QR Code

A API retorna o QR Code diretamente na resposta do `/checkout`:

```typescript
// Exemplo de implementação React
function CheckoutSuccess({ response }: { response: CheckoutResponse }) {
  if (response.status === "ACTIVE") {
    return <SuccessMessage>Pagamento aprovado! ✅</SuccessMessage>;
  }
  
  if (response.pix_code) {
    return (
      <PixPaymentScreen>
        <h2>Escaneie o QR Code para pagar</h2>
        
        {/* QR Code Image */}
        <img 
          src={response.pix_qr_code_url} 
          alt="QR Code PIX"
          className="w-64 h-64 mx-auto"
        />
        
        {/* Copia e Cola */}
        <div className="mt-4">
          <label>Ou copie o código PIX:</label>
          <div className="flex gap-2">
            <input 
              readOnly 
              value={response.pix_code}
              className="flex-1 font-mono text-xs"
            />
            <button onClick={() => copyToClipboard(response.pix_code)}>
              Copiar
            </button>
          </div>
        </div>
        
        {/* Info */}
        <Alert severity="info">
          O PIX expira em 30 minutos. Após o pagamento, sua assinatura 
          será ativada automaticamente.
        </Alert>
        
        {/* Polling Status */}
        <StatusChecker customerId={response.id} />
      </PixPaymentScreen>
    );
  }
  
  return <ErrorMessage>Erro ao gerar pagamento</ErrorMessage>;
}

// Component para verificar status (polling)
function StatusChecker({ customerId }: { customerId: string }) {
  const [status, setStatus] = useState("PENDING");
  
  useEffect(() => {
    const interval = setInterval(async () => {
      const res = await fetch(`/customers/${customerId}/status`);
      const data = await res.json();
      setStatus(data.status);
      
      if (data.status === "ACTIVE") {
        clearInterval(interval);
        // Redirecionar para página de sucesso
        router.push("/sucesso");
      }
    }, 5000); // Verifica a cada 5 segundos
    
    return () => clearInterval(interval);
  }, [customerId]);
  
  return (
    <div className="text-center mt-4">
      {status === "PENDING" ? (
        <Spinner text="Aguardando pagamento..." />
      ) : (
        <SuccessIcon />
      )}
    </div>
  );
}
```

---

## 💡 Recomendações de UX

### 1. **Validações no Frontend**
```typescript
function validateDependent(dep: DependentInput): string[] {
  const errors: string[] = [];
  
  if (!dep.name || dep.name.length < 3) {
    errors.push("Nome deve ter ao menos 3 caracteres");
  }
  
  if (!isValidCPF(dep.cpf)) {
    errors.push("CPF inválido");
  }
  
  const age = calculateAge(dep.birth_date);
  if (age < 0 || age > 120) {
    errors.push("Data de nascimento inválida");
  }
  
  return errors;
}
```

### 2. **Auto-save no LocalStorage**
```typescript
// Salvar automaticamente enquanto o usuário preenche
useEffect(() => {
  if (dependents.length > 0) {
    localStorage.setItem('checkout_dependents', JSON.stringify(dependents));
  }
}, [dependents]);

// Restaurar ao voltar
useEffect(() => {
  const saved = localStorage.getItem('checkout_dependents');
  if (saved) {
    setDependents(JSON.parse(saved));
  }
}, []);
```

### 3. **Visual Feedback**
```tsx
{dependents.length > 0 && (
  <Alert severity="success">
    ✅ {dependents.length} dependente(s) será(ão) incluído(s) no plano
  </Alert>
)}
```

### 4. **Limite Máximo (Opcional)**
```typescript
const MAX_DEPENDENTS = 5;

function addDependent() {
  if (dependents.length >= MAX_DEPENDENTS) {
    toast.error(`Máximo de ${MAX_DEPENDENTS} dependentes permitido`);
    return;
  }
  // ...
}
```

### 5. **Confirmação Antes de Remover**
```typescript
function handleRemove(index: number) {
  const dep = dependents[index];
  if (confirm(`Remover ${dep.name} dos dependentes?`)) {
    setDependents(prev => prev.filter((_, i) => i !== index));
  }
}
```

---

## 🚀 Exemplo Completo de Integração

```typescript
// hooks/useCheckout.ts
export function useCheckout() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  async function submitCheckout(data: CheckoutRequest) {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch('/checkout', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      
      if (!response.ok) {
        const err = await response.json();
        throw new Error(err.message || 'Erro ao processar checkout');
      }
      
      const result: CheckoutResponse = await response.json();
      return result;
    } catch (err) {
      setError(err.message);
      throw err;
    } finally {
      setLoading(false);
    }
  }
  
  return { submitCheckout, loading, error };
}

// components/CheckoutForm.tsx
export function CheckoutForm() {
  const { submitCheckout, loading } = useCheckout();
  const [formData, setFormData] = useState<CheckoutRequest>({
    // ... campos do titular
    dependents: [],
  });
  
  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    
    try {
      const result = await submitCheckout(formData);
      
      // Se for PIX, mostrar QR Code
      if (result.pix_code) {
        router.push({
          pathname: '/pagamento/pix',
          query: { 
            code: result.pix_code,
            qr: result.pix_qr_code_url,
            customerId: result.id 
          }
        });
      } else {
        // Cartão aprovado direto
        router.push('/sucesso');
      }
    } catch (err) {
      toast.error('Erro ao processar pagamento');
    }
  }
  
  return (
    <form onSubmit={handleSubmit}>
      {/* Campos do titular */}
      
      <DependentsSection 
        dependents={formData.dependents}
        onChange={(deps) => setFormData(prev => ({ ...prev, dependents: deps }))}
      />
      
      <Button type="submit" loading={loading}>
        {formData.payment_method === 'PIX' ? 'Gerar QR Code' : 'Finalizar Pagamento'}
      </Button>
    </form>
  );
}
```

---

## ✅ Checklist de Implementação

- [ ] Adicionar campo `dependents` opcional no formulário
- [ ] Validar CPF e data de nascimento dos dependentes
- [ ] Implementar tela de PIX com QR Code + Copia e Cola
- [ ] Polling a cada 5s para verificar status do pagamento
- [ ] Limpar dados do localStorage após sucesso
- [ ] Adicionar loading states e error handling
- [ ] Testar com 0, 1 e múltiplos dependentes
- [ ] Testar fluxo PIX vs Cartão

---

## 🎯 Recomendação Final

**Use a Opção 1 (Toggle)** se:
- Seu formulário já é longo
- Quer simplicidade máxima
- A maioria dos usuários NÃO terá dependentes

**Use a Opção 2 (Step Dedicado)** se:
- Já tem multi-step form
- Quer dar destaque à feature
- Espera que muitos usuários adicionem dependentes

**Sobre o QR Code:**
- ✅ Já funciona - QR Code vem na resposta do /checkout
- ⚠️ Implementar polling para verificar quando pagamento é confirmado
- 💡 Considere criar rota `/pagamento/pix/[customerId]` no frontend para poder compartilhar link
