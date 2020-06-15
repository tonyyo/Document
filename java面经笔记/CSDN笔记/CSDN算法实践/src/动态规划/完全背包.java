package 动态规划;

public class 完全背包 {
    public static void main(String[] args) {
        int[] weight = {2, 3, 4, 5};
        int[] value = {3, 4, 5, 6};
        int N = 4;
        int capacity = 8;
        int[] beibao = new int[capacity + 1]; // 数组默认元素都是0, 背包容量始终加1
        for (int i = 0; i < N; i++) {
            for (int j = weight[i]; j <= capacity ; j++) {  // 从能放入的物体的体积作为背包的起始容量
                if (weight[i] <= j){
                    beibao[j] = Math.max(beibao[j], beibao[j - weight[i]] + value[i]);
                }
            }
        }

        System.out.println(beibao[capacity]);
    }
}
