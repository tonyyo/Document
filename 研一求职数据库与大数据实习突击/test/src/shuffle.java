import java.util.ArrayList;
import java.util.List;
import java.util.Random;

public class shuffle {
    public static void main(String[] args) {
        ArrayList<String> list = new ArrayList<String>();
        Random random = new Random();
        list.add("1");
        list.add("2");
        list.add("3");
        list.add("4");
        list.add("5");
        for (int i = list.size() - 1; i > 0; i--) {
            swap(list, i, random.nextInt(i));  // 一个一个位置打乱顺序，跟前面的数进行交换，后面数不变，能够保证所有的数都不在原来的那个位置上
        }
        for (String str :
                list) {
            System.out.println(str);
        }

    }
    static void swap(List<String> list, int first, int sec){
        String temp = list.get(first);
        list.set(first, list.get(sec));
        list.set(sec, temp);
    }
}
