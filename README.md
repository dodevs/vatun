# Vatun
Vatun (valida túnel) é um programa para criar túneis de layer 4 (camada de transporte).

## Como compilar
O programa foi desenvolvido na linguagem Go. Antes de qualquer coisa, faça o download do compilador e instale conforme o especificado no site oficial: https://golang.org/dl/  

OBS.: O programa foi compilado e testado com as versões 1.11 e 1.9  

A biblioteca para programar as interfaces tuntap foi ligeiramente modificada, para que seja possível utilizar as interfaces do tipo TUN. Logo, evite usar a versão oficial.  

Para gerar o executável, siga os passos:
```
git clone https://git.pop-es.rnp.br/matheus.garcias/vatun.git
cd vatun
./build.sh
```
Ou ainda:
```
git clone https://git.pop-es.rnp.br/matheus.garcias/vatun.git
cd vatun
go build -o vatun cmd/*.go
```

## Como usar
Para usar o programa basta executar o executável passando os argumentos necessários.  

As opções são as seguintes:  
*   ```-ip=<IP:PORTA>```  
        Especifica o ip e porta. Caso esteja no modo servidor, escuta nesse ip e porta. Caso seja modo cliente conecta nesse ip e porta.
*   ```-mtu=<número inteiro>```  
        Especifica o MTU da interface criada para o túnel.
*   ```-s```   
        Funciona no modo servidor.
*   ```-c```  
        Funciona no modo cliente.  

### Exemplo:
IP do servidor: 192.168.0.2  
IP do cliente: 192.168.0.3

Lado do servidor:
```
./vatun -s -ip=0.0.0.0:8080 -mtu=1500
```
O programa fica escutando na porta 8080 por novas requisições de conexão.  

Lado do cliente:
```
./vatun -c -ip=192.168.0.2 -mtu=1500
```
O programa faz uma requisição de conexão no IP e porta do servidor.  
Caso a conexão seja feita com sucesso, as interfaces já estarão <i>UP</i> e com o <i>MTU</i> configurado.  
No momento, para fins de testes, a configuração da rede é um ponto a ponto. Para testar a conexão faça:  
No servidor:
```
ping 10.253.0.2
```
No cliente:
```
ping 10.253.0.1
```