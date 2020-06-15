//*****************************************************************
// server.java 
//
// Allows clients to connect and request files.  If the file
// exists it sends the file to the client.  
//*****************************************************************

import java.lang.*;
import java.io.*;
import java.net.*;
import java.util.Scanner;

public class server
{
    private final static int PORT = 12345;
    private final static int QUEUE_SIZE = 10;
    private final static int BUF_SIZE = 4096;

    public static void main(String[] args)
    {
        try {
            // Create the server socket
            ServerSocket sSocket = new ServerSocket(PORT, QUEUE_SIZE);

            // Socket is set up and will wait for connections
            while (true) {  //不用true会好一些
                // Listen for a connection to be made to this socket and accept it
                Socket cSocket = sSocket.accept();
                byte[] byteArray = new byte[BUF_SIZE];

                // Get the name of the file from the client
                Scanner scn = new Scanner(cSocket.getInputStream());
                String fileName = scn.next();

                // Send the contents of the file
                BufferedInputStream bis = new BufferedInputStream(new FileInputStream(fileName)); //这里的输入流是文件里面的，所以要用文件输入流
                OutputStream outStream = cSocket.getOutputStream();
                while(bis.available() > 0) {
                    bis.read(byteArray, 0, byteArray.length);
                    outStream.write(byteArray, 0, byteArray.length);
                }

                // Close
                bis.close();
                cSocket.close();
            }
        }
        catch(EOFException eofe) {
            eofe.printStackTrace();
            System.exit(1);
        }
        catch(FileNotFoundException fnfe) {
            fnfe.printStackTrace();
            System.exit(1);
        }
        catch(IllegalArgumentException iae) {
            iae.printStackTrace();
            System.out.println("Bind failed");
            System.exit(1);
        }
        catch(IOException ioe) {
            ioe.printStackTrace();
            System.out.println("Could not complete request");
            System.exit(1);
        }
    }
}