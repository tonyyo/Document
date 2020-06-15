//*****************************************************************
// client.java 
//
// Connects to the server and sends a request for a file by 
// the file name.  Prints the file contents to standard output.  
//*****************************************************************

import java.lang.*;
import java.io.*;
import java.net.*;

public class client
{
    private final static int PORT = 12345;
    private final static int BUF_SIZE = 4096;

    public static void main(String[] args)
    {
        // Set up socket using host name and port number
        try {
            // Get host name and file name from command line arguments
            String host = args[0];
            String fileName = args[1];

            Socket s = new Socket(host, PORT);
            byte[] byteArray = new byte[BUF_SIZE];
            // Send filename to the server
            PrintWriter pw = new PrintWriter(s.getOutputStream(), true);
            pw.println (fileName);

            // Get the file from the server and print to command line
            DataInputStream fromServer = new DataInputStream(s.getInputStream());

            while(fromServer.read(byteArray) > 0) {
                System.out.println(new String(byteArray, "UTF-8"));
            }

            fromServer.close();
            s.close();
        }
        catch(IndexOutOfBoundsException iobe) {
            System.out.println("Usage: client host-name file-name");
            System.exit(1);
        }
        catch(UnknownHostException unhe) {
            unhe.printStackTrace();
            System.out.println("Unknown Host, Socket");
            System.exit(1);
        }
        catch(IOException ioe) {
            ioe.printStackTrace();
            System.exit(1);
        }
    }
}