from pathlib import Path
import re

path = Path('/Users/carloseduardosilvaxavier/Documents/projects/ligue-payments/teste.html')
text = path.read_text()

text, c1 = re.subn(r'\n\s*<div class="d-flex justify-content-end mb-3">\n\s*<button id="btnDemo" class="btn btn-sm btn-dark fw-bold">⚡ Preencher Demo</button>\n\s*</div>', '', text, count=1)
text, c2 = re.subn(r"function updateStep3Layout\(\) \{\n\s*if \(finalSubmitBtn\) finalSubmitBtn\.textContent = 'Finalizar Pedido';\n\s*validateStep3\(\);\n\s*\}",
                   "function updateStep3Layout() {\n            if (finalSubmitBtn) {\n                var isPix = paymentMethodInput && paymentMethodInput.value === 'PIX';\n                finalSubmitBtn.textContent = 'Finalizar Pedido';\n                finalSubmitBtn.classList.toggle('d-none', isPix);\n            }\n            validateStep3();\n        }",
                   text, count=1)
text, c3 = re.subn(r"function showActiveCpfModal\(\) \{\n\s*var msg = 'Opa, vimos que você já é de casa! 🩵 Usuário já cadastrado\. Para dúvidas sobre o seu plano ou atualizações, contate nosso SAC\.';\n\s*if \(activeCpfModalMessage\) \{\n\s*activeCpfModalMessage\.textContent = msg;\n\s*\}\n\s*if \(activeCpfModal\) \{\n\s*activeCpfModal\.show\(\);\n\s*\} else \{\n\s*alert\(msg\);\n\s*\}\n\s*\}",
                   "function showActiveCpfModal() {\n            var msg = 'Que bom te ver por aqui de novo 💚 Você já é de casa! O CPF informado já tem cadastro ativo, então não precisa seguir para o pagamento.';\n            if (activeCpfModalMessage) {\n                activeCpfModalMessage.textContent = msg;\n            }\n            if (activeCpfModal) {\n                activeCpfModal.show();\n            } else {\n                alert(msg);\n            }\n        }",
                   text, count=1)
text, c4 = re.subn(r"\n\s*document\.getElementById\('btnDemo'\)\.addEventListener\('click', function \(\) \{.*?\n\s*\}\);\n", '\n', text, count=1, flags=re.S)

path.write_text(text)
print({'demo_removed': c1, 'step3_updated': c2, 'modal_updated': c3, 'demo_listener_removed': c4})
