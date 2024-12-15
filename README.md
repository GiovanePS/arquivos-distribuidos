# Implementação de um serviço de transferência de arquivos em rede

Este serviço tem funcionalidade semelhante a dos comandos scp e FTP, por exemplo, onde um
comando executado permite que um cliente envie (receba) um arquivo para (de) um nodo
remoto. Para a comunicação distribuída, serão utilizadas APIs de comunicação que
implementam Berkeley Sockets.

## Funcionalidades e peculiaridades

### O exemplo a seguir ilustra o uso do comando remcp (acrônimo para remote copy), que será implementado.
```bash
$ ./remcp meu_arquivo.txt 192.168.0.5:/home/usuario/teste

$ ./remcp 192.168.0.5:/home/usuario/teste/arq1.png /tmp/
```
Na primeira linha, o arquivo meu_arquivo.txt é enviado do diretório
corrente para o nodo remoto 192.168.0.5 e é salvo no diretório /home/usuario/teste.
Na segunda linha, o comando remcp envia o arquivo arq1.png localizado no diretório
/home/usuario/teste/ para a pasta /tmp do nodo remoto. Para a identificação dos
caminhos de origem e destino, assuma a representação de caminhos de diretório e arquivos
utilizados em sistemas baseados em UNIX.

### O comando remcp permite a parada e reinicialização da transferência de arquivo.
No caso de falhas ou interrupção do serviço durante o uso a cada ciclo de transferência, o comando
cria ou atualiza um arquivo temporário, representado pelo nome do arquivo seguido pela
extensão .part. Por exemplo, o arquivo arq1.png manteria a cópia parcial do conteúdo
recebido no arquivo arq1.png.part,o qual seria criado após receber os primeiros bytes e
atualizado com os bytes em sequência, até que não tenha mais conteúdo para transferir.
Ao completar a transferência, o arquivo nome arq1.png.part é renomeado para nome arq1.png.
Por padrão, o arquivo é salvo no diretório /tmp de sistemas Unix.

Ao iniciar uma transferência, o serviço verifica se há uma cópia parcial do arquivo neste diretório.
Caso encontre o respectivo arquivo .part, o serviço verifica o tamanho do arquivo parcial e
transfere apenas o conteúdo restante.

### O servidor pode atender múltiplos clientes simultaneamente.
Para evitar sobrecarga e monopolização da rede, o servidor transfere arquivos respeitando uma taxa de
transferência limitada e configurável, dada em bytes/s. Por exemplo, se a taxa de
transferência for de 256 bytes por segundo, o servidor não pode enviar mais do que 256
bytes por segundo. Se o servidor estiver atendendo mais de um cliente simultaneamente, a taxa
de transferência deve ser dividida, de modo a permitir transferências simultâneas sem
exceder a taxa de transferência.

### O servidor atende um número máximo de transferências simultâneas.
No caso de um cliente tentar transferir um arquivo de/para um servidor que esteja operando no seu limite, o
servidor retorna uma mensagem de erro e o cliente irá tentar retomar a transferência novamente
