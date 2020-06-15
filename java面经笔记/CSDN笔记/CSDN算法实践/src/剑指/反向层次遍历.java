package 剑指;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedList;

public class 反向层次遍历 {
    public static void main(String[] args) {
        int[] ints = new int[]{1, 2, 3, 4, 5, 6, 7};
        ArrayList<Integer> list = new ArrayList<>();
        for (int i :
                ints) {
            list.add(i);
        }
        TreeNode root = buildBanlanceTree(list, 0, list.size() - 1);
        ArrayList<Integer> resultList = PrintFromTopToBottom(root);
        for (int x :
                resultList) {
            System.out.print(x + " ");
        }
    }

    static TreeNode buildBanlanceTree(ArrayList<Integer> list, int start, int end){ //构建平衡二叉树
        if (start == end){
            return new TreeNode(list.get(start));
        }
        int mid = (start + end) / 2;
        TreeNode root = new TreeNode(list.get(mid));
        root.left = buildBanlanceTree(list, start, mid - 1);
        root.right = buildBanlanceTree(list, mid + 1, end);
        return root;
    }

    static ArrayList<Integer> PrintFromTopToBottom(TreeNode root) {
        LinkedList<TreeNode> queue = new LinkedList<>();
        ArrayList<Integer> list = new ArrayList<>();
        queue.add(root);
        while (!queue.isEmpty()){
            ArrayList<Integer> level = new ArrayList<>();
            int levelNum = queue.size();  // 当前层的节点数
            for (int i = 0; i < levelNum; i++) {
                TreeNode pos = queue.pollFirst();
                level.add(pos.val);
                if (pos.left != null)
                    queue.add(pos.left);
                if (pos.right != null)
                    queue.add(pos.right);
            }
            Collections.reverse(level);
            list.addAll(level);
        }
        return list;
    }
}
