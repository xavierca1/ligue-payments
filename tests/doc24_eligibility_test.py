"""
Teste de elegibilidade Doc24 — envio de uma vida via JSON

Uso:
    pip install requests
    python tests/doc24_eligibility_test.py
"""

import json
import requests

BASE_URL = "api_de_homologacao_doc24"  # Substitua pela URL real da API de homologação do Doc24, certificando-se de que a URL esteja correta e acessível para realizar os testes de elegibilidade.
AUTH_URL = f"{BASE_URL}/authentication"
ELEG_URL = f"{BASE_URL}/portal/elegibilidad"

CLIENT_ID     = "insira_seu_client_id_aqui"  # Substitua pelo seu client_id real
CLIENT_SECRET = "insira_seu_client_secret_aqui"  # Substitua pelo seu client_secret real

# Array para teste de elegibilidade, com dados fictícios. Substitua pelos dados reais do afiliado que deseja testar. Certifique-se de que os dados estejam corretos e completos para obter um resultado preciso da elegibilidade.
AFILIADO = {
    "nro_documento":         "Insira_o_número_de_documento_aqui",  # Substitua pelo número de documento real do afiliado (CPF ou outro tipo de documento, conforme exigido pela API)
    "credencial":            "Insira_a_credencial_aqui",  # Substitua pela credencial real do afiliado, se aplicável como CPF
    "apellido":              "Xavier",
    "fecha_alta":            "2026-05-18",
    "fecha_nacimiento":      "2001-08-24",
    "nro_documento_titular": "Insira_o_número_do_documento_do_titular_aqui",  # Substitua pelo número de documento do titular, se aplicável
    "sexo":                  "M",
    "empresa":               "Ligue_digital",
    "nombre":                "Charles",
    "plan":                  "ligue vida plena individual", # Os nomes dos planos estao disponiveis na tabela plans no supabase em plan_name_provider, ou podem ser consultados via API de planos
    "telefono_movil":        "61999999999 ",  # Substitua pelo número de telefone móvel com DDD real do afiliado
    "email":                 "teste@exemplo.com",  # Substitua pelo email real do afiliado
}


#Funcao de autenticação para obter o token de acesso necessário para fazer a requisição de elegibilidade. Certifique-se de que as credenciais (CLIENT_ID e CLIENT_SECRET) estejam corretas e tenham permissão para acessar a API do Doc24.
def authenticate() -> str:
    resp = requests.post(AUTH_URL, json={"client_id": CLIENT_ID, "client_secret": CLIENT_SECRET})
    resp.raise_for_status()
    return resp.json()["token"]


def main():
    token = authenticate()

#Requisicao que cria ou cadastra o paciente (vida) na base da Doc24 e retorna a elegibilidade do mesmo. Certifique-se de que os dados do afiliado estejam corretos e completos para obter um resultado preciso da elegibilidade. O resultado da elegibilidade será impresso no console em formato JSON, facilitando a leitura e análise dos dados retornados pela API.
    resp = requests.post(
        ELEG_URL,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        },
        json={"afiliado": AFILIADO},
    )

    print(json.dumps(resp.json(), indent=4, ensure_ascii=False)) 
    # Expect a seguinte resposta JSON, onde "estado" indica a elegibilidade (1 para elegível, 0 para não elegível) e "mensaje" fornece uma descrição do resultado: 
    # {
#     "estado": 1,
#     "mensaje": "OK"
# }


if __name__ == "__main__":
    main()
