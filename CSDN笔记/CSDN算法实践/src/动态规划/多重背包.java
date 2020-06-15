package 动态规划;

public class 多重背包 {
    public static void main(String[] args) {
        int[] weight = {2, 3, 4, 5};
        int[] value = {3, 4, 5, 6};
        int[] count = {2, 2, 2, 2};  // 每个物品的数量
        int N = 4;
        int capacity = 8;
        int[] beibao = new int[capacity + 1]; // 数组默认元素都是0, 背包容量始终加1
        for (int i = 0; i < N; i++) {
            for (int j = capacity; j >= weight[i] ; j--) {  // 从能放入的物体的体积作为背包的起始容量
                int num = Math.min(count[i], capacity / weight[i]); // 能放入背包的该物体的最大数量
                for (int k = 1; k <= num ; k <<= 1) {  // k从1开始，每次乘以2，直到大于该物体的数量或大于背包容量
                    if (k * weight[i] <= j){
                        beibao[j] = Math.max(beibao[j], beibao[j - k *  weight[i]] + k *  value[i]);
                    }
                }
            }
        }

        System.out.println(beibao[capacity]);
    }
}
