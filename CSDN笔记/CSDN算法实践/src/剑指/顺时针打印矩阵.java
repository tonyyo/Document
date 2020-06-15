package 剑指;

import java.security.cert.TrustAnchor;
import java.util.ArrayList;
import java.util.LinkedList;

public class 顺时针打印矩阵 {
    public static void main(String[] args) {
        int[][] matrix = new int[][]{
                {1, 2, 3, 4},
                {5, 6, 7, 8},
                {9, 10, 11, 12},
                {13, 14, 15, 16}};
        int[] dx = new int[]{0, 1, 0, -1};
        int[] dy = new int[]{1, 0, -1, 0};
        int x = 0, y = 0;
        int direct = 0, rowNum = matrix.length, colNum = matrix[0].length;
        ArrayList<Integer> list = new ArrayList<>(); // record path
        int[][] visit = new int[rowNum][colNum];  // record visited point
        list.add(matrix[0][0]);
        visit[0][0] = 1;
        for(int i = 1; i < rowNum * colNum; i ++){
            int nextX = x + dx[direct], nextY = y + dy[direct];
            if (!outbound(matrix, nextX, nextY) || visit[nextX][nextY] == 1){
                if (direct == 0){// 0 : right; if outbound:  set direct = 1
                    x = x + 1;
                    direct = 1;
                }
                else if (direct == 1){
                    y = y - 1;
                    direct = 2;
                }
                else if (direct == 2){
                    x = x - 1;
                    direct = 3;
                }
                else {
                    y = y + 1;
                    direct = 0;
                }
            }
            else{
                x = nextX;
                y = nextY;
            }
            list.add(matrix[x][y]);
            visit[x][y] = 1;
        }
        for (Integer i :
                list) {
            System.out.print(String.valueOf(i) + " ");
        }
    }

    static boolean outbound(int[][] matrix, int x, int y){
        int rowNum = matrix.length, colNum = matrix[0].length;
        if (!(0 <= x && x < rowNum && 0 <= y && y < colNum))
            return false;
        else
            return true;
    }

}
