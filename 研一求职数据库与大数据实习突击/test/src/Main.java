
public class Main {
    public static void main(String[] args) {
        String str = "We Are Happy";
        StringBuffer stringBuffer = new StringBuffer(str);
        String string = stringBuffer.toString();
        String s = string.replaceAll(" ", "%20");
        System.out.println(s);
        String s1 = new String("123");
    }
}
