package 动态规划;

public class 数字串转化为字母串 {
    public static void main(String[] args) {
        String str = "234";
        int N = str.length();
        int a = 1;
        int b = 2;
        if(Integer.valueOf(str.substring(0, 2)) > 26){
            b = 1;
        }
        for (int i = 2; i < N; i++) {
            if(Integer.valueOf(str.substring(i-1, i+1)) > 26){
                int temp = b;  // 只有一种情况
                a = b;
                b = temp;
            }
            else{
                int temp = a + b;
                a = b;
                b = temp;
            }
        }
        System.out.println(b);
    }
}
