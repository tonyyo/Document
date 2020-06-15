package 剑指;

import sun.font.TrueTypeFont;

import java.util.ArrayList;
import java.util.LinkedList;

public class 栈的压入_弹出序列 {
    public static void main(String[] args) {
        int[] ints = new int[]{1, 2, 3, 4, 5};
        int[] out1 = new int[]{4, 5, 3, 2, 1};
        int[] out2 = new int[]{4, 3, 5, 1, 2};
        System.out.println(IsPopOrder(ints, out1));
    }
    static boolean IsPopOrder(int [] pushA, int [] popA) {
        LinkedList<Integer> list = new LinkedList<>(); // 辅助栈
        int i = 0, j = 0;
        while (j < pushA.length){
            if (pushA[j] != popA[i]){
                list.add(pushA[j]);
                j++;
            }
            else{
                i++;
                j++;
            }
        }
        if (list.isEmpty())
            return true;
        while( i < popA.length ){
            if (popA[i] != list.pollLast())
                return false;
            i++;
        }
        return true;
    }
}
