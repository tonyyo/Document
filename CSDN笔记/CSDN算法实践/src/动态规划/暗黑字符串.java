package 动态规划;

import java.util.ArrayList;

public class 暗黑字符串 {
    public static void main(String[] args) {
        System.out.println(anhei(3));
    }
    static int anhei(int n){
        int[] dp = new int[n + 1]; // dp[n]: n个长度的字符串含暗黑字符串的个数
        dp[0] = 0;
        dp[1] = 3;
        dp[2] = 9;  // 前面必须有两个元素，才能分情况讨论
        for (int i = 3; i <= n ; i++) {
            dp[i] = 2 * dp[i - 1] + dp[i - 2];
        }
        return dp[n];
    }
}
