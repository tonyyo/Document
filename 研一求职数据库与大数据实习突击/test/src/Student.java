import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.File;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;

public class Student {

    public static void main(String[] args)  {
        // TODO Auto-generated method stub
        //定义字符串数组
        String content[] = {"枯藤老树昏鸦","小桥流水人家","古道西风瘦马","夕阳西下","断肠人在天涯"};
        File file = new File("F:/std.txt");   //创建文件对象
        while(!file.exists()) {
            try {                     //捕捉异常
                file.createNewFile(); //如果此文件不存在则新建此文件 ，此处有异常需要处理
                System.out.println("新文件已创建！");
            }catch(Exception e){      //处理异常
                e.printStackTrace();   //输出异常
            }
        }
        try {             //捕捉异常
            FileWriter w = new FileWriter(file);  //此处有异常应该被处理
            BufferedWriter  bfw = new BufferedWriter(w); //创建BufferedWriter类对象
            for(int i=0;i<content.length;i++) {
                bfw.write(content[i]);     //使用for循环依次往文件中输入数组中的元素
                bfw.newLine();   //写入一个行分隔符
            }

            bfw.close(); //关闭BufferedWriter流
            w.close();   //关闭FileWriter流
        }catch(Exception e) {   //处理异常
            e.printStackTrace();  //输出异常
        }
        System.out.println("数据已写入std.txt文件");
        //-------------下面开始读出数据-------------------
        try {
            FileReader r = new FileReader(file);  //创建FileReader类对象，此处有异常需要处理
            BufferedReader bfr = new BufferedReader(r);
            int i = 0;
            String s = null;
            while((s=bfr.readLine())!=null) {  //如果文件的文本行数不为null，则进入循环
                i++;
                System.out.println("第"+i+"行:"+s);
            }
            bfr.close();
            r.close();
        }catch(Exception e) {
            e.printStackTrace();
        }


    }

}