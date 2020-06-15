package 剑指;

public class test {
    public static void main(String[] args) {
        int[] ints = {1, 2, 3};
        int[] temp = ints.clone();
        temp[2] = 4;
        System.out.println(ints[2]);
    }
}
