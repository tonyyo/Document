package 动态规划;

public class 零壹背包 {
    public static void main(String[] args) {
        int[] weight = {2, 3, 4, 5};
        int[] value = {3, 4, 5, 6};
        int N = 4;
        int capacity = 8;
        int[] beibao = new int[capacity + 1]; // 数组默认元素都是0, 背包容量始终加1
        for (int i = 0; i < N; i++) {
            for (int j = 8; j >= weight[i] ; j--) {  // 从能放入的物体的体积作为背包的起始容量
                if (weight[i] <= j){
                    beibao[j] = Math.max(beibao[j], beibao[j - weight[i]] + value[i]);
                }
            }
            for (int x :
                    beibao) {
                System.out.print(x + " ");
            }
            System.out.println();
        }
        System.out.println(beibao[capacity]);
    }
}
